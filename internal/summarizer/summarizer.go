package summarizer

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/soulteary/gleaner/internal/nnmodel"
	"github.com/soulteary/gleaner/internal/segment"
	"github.com/soulteary/gleaner/internal/similarity"
	"github.com/soulteary/gleaner/internal/textrank"
)

var summaryClearRules = []struct {
	re  *regexp.Regexp
	rep string
}{
	{regexp.MustCompile(`^编者按：`), ""},
	{regexp.MustCompile(`^[^，日天前]{1,4}，`), ""},
	{regexp.MustCompile(`^“`), ""},
	{regexp.MustCompile(`^其该此而[^，]{1,10}，`), ""},
	{regexp.MustCompile(`^[并且至于而]+`), ""},
}

var sentenceSplitRe = regexp.MustCompile(`(?:\n|。|？|！)`)

// Summarizer generates extractive summaries.
type Summarizer struct {
	seg     segment.Segmenter
	model   *nnmodel.Model
	shingle bool
	filter  []string
}

// New creates a summarizer.
func New(seg segment.Segmenter, model *nnmodel.Model, shingle bool, filter []string) *Summarizer {
	return &Summarizer{
		seg:     seg,
		model:   model,
		shingle: shingle,
		filter:  filter,
	}
}

// Summarize generates a summary of expected length from content.
func (s *Summarizer) Summarize(content string, expectedLength float64, title string) string {
	terms := s.splitToTerms(content)
	var sentences [][]string

	var spacesTitle string
	if title != "" {
		spacesTitle = strings.Join(s.seg.Cut(strings.TrimSpace(title)), " ")
	}

	for x := 0; x < len(terms); x++ {
		for y := 0; y < len(terms[x]); y++ {
			if s.shingle {
				var sentence []string
				cuSize := 0
				maxCovered := y
				for z := y; z < len(terms[x]); z++ {
					cuSize += len(terms[x][z])
					if z > y && float64(cuSize) > expectedLength {
						break
					}
					sentence = append(sentence, terms[x][z]...)
					maxCovered = z
				}
				sentences = append(sentences, sentence)
				if maxCovered >= len(terms[x])-1 {
					break
				}
			} else {
				sentences = append(sentences, terms[x][y])
			}
		}
	}

	if len(sentences) == 0 {
		return ""
	}

	graph := buildSimilarityGraph(sentences)
	textRank := textrank.PageRank(graph, 0.85, 0.0001)

	type ranked struct {
		score float64
		idx   int
	}
	copiedRank := make([]ranked, len(sentences))
	lenArr := make([]int, len(sentences))
	maxLen := 0
	for i, sent := range sentences {
		lenArr[i] = len(sent)
		if lenArr[i] > maxLen {
			maxLen = lenArr[i]
		}
	}

	for i := range sentences {
		originSentence := strings.Join(sentences[i], "")
		spacesSentence := strings.Join(sentences[i], " ")
		titleSim := 0.1
		if spacesTitle != "" {
			titleSim = similarity.CompareTwoStrings(spacesTitle, spacesSentence)
		}
		lenScore := float64(len(sentences[i])) / float64(maxLen)
		modelScore := s.model.Rank(s.seg, originSentence)
		finalScore := textRank[i]*0.5 + lenScore*0.2 + modelScore*0.2 + titleSim*0.1
		copiedRank[i] = ranked{score: finalScore, idx: i}
	}

	sort.Slice(copiedRank, func(i, j int) bool {
		return copiedRank[i].score > copiedRank[j].score
	})

	var selectIndex []int
	summarySize := 0
	for i := 0; i < len(copiedRank); i++ {
		ordinal := copiedRank[i].idx
		if s.isValidateSentence(sentences[ordinal]) {
			summarySize += len(sentences[ordinal])
			if len(selectIndex) > 0 {
				if float64(summarySize) > expectedLength {
					break
				}
				selectIndex = append(selectIndex, ordinal)
			} else {
				selectIndex = append(selectIndex, ordinal)
			}
		}
	}

	sort.Ints(selectIndex)
	var selectSentences []string
	for _, i := range selectIndex {
		selectSentences = append(selectSentences, strings.Join(sentences[i], ""))
	}
	ret := strings.Join(selectSentences, "...")
	for _, rule := range summaryClearRules {
		ret = rule.re.ReplaceAllString(ret, rule.rep)
	}
	return ret
}

func (s *Summarizer) splitToTerms(content string) [][][]string {
	paragraphs := strings.Split(content, "\n\n")
	var terms [][][]string
	for _, p := range paragraphs {
		sens := sentenceSplitRe.Split(p, -1)
		var parag [][]string
		for _, y := range sens {
			y = strings.TrimSpace(y)
			if y == "" {
				continue
			}
			cut := s.seg.Cut(y)
			var sentens []string
			for _, z := range cut {
				sentens = append(sentens, z)
			}
			if len(sentens) > 0 {
				last := sentens[len(sentens)-1]
				if last != "。" {
					sentens = append(sentens, "。")
				}
				parag = append(parag, sentens)
			}
		}
		if len(parag) > 0 {
			terms = append(terms, parag)
		}
	}
	return terms
}

func isASCIIAlphaNumeric(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func (s *Summarizer) isValidateSentence(charArr []string) bool {
	if len(charArr) == 0 {
		return false
	}
	slen, dlen := 0, 0
	text := strings.Join(charArr, "")
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			// 空白不计入任何一侧，避免噪声干扰判定
		case isASCIIAlphaNumeric(r) || (r >= 0xff60 && r <= 0xff9f):
			// 仅统计字母/数字类 ASCII（含半角片假名），排除标点噪声
			slen++
		case r >= 0x0001 && r <= 0x007e:
			// 其余 ASCII（标点/符号）既不计中文也不计 ASCII 主体
		default:
			dlen++
		}
	}
	// 仅当句子里没有任何中文（dlen==0）且 ASCII 字母数字占主导时，才判为非中文句而丢弃；
	// 含少量英文/数字（如公司名、型号）的中文句不再被误删。
	if dlen == 0 && slen > 0 {
		return false
	}
	if len(s.filter) > 0 {
		for _, rule := range s.filter {
			if matched, _ := regexp.MatchString("(?i)"+rule, text); matched {
				return false
			}
		}
	}
	return true
}

// buildSimilarityGraph produces a normalized similarity matrix for TextRank.
// Levenshtein distance is computed on rune-level and converted to a similarity
// weight (1 - dist/maxLen) so that "more similar" yields a larger value, matching
// TextRank semantics. Self-loops (i==j) are set to 0 to avoid dominating the graph.
func buildSimilarityGraph(sentences [][]string) [][]float64 {
	n := len(sentences)
	graph := make([][]float64, n)
	joined := make([]string, n)
	for i := 0; i < n; i++ {
		joined[i] = strings.Join(sentences[i], ",")
	}
	for i := 0; i < n; i++ {
		graph[i] = make([]float64, n)
		li := len([]rune(joined[i]))
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			lj := len([]rune(joined[j]))
			maxLen := li
			if lj > maxLen {
				maxLen = lj
			}
			if maxLen == 0 {
				continue
			}
			dist := textrank.Distance(joined[i], joined[j])
			graph[i][j] = 1 - float64(dist)/float64(maxLen)
		}
	}
	return graph
}

// ResolveExpectedLength handles ratio-based length (0 < exptd < 1) for Summarize API.
// Matches legacy summarize(): parseInt(blocks.join(”).length * exptd_len).
func ResolveExpectedLength(blocks []string, exptdLen float64, minLen, maxLen int) float64 {
	if exptdLen > 0 && exptdLen < 1 {
		plain := strings.Join(blocks, "")
		length := int(float64(len(plain)) * exptdLen) // parseInt semantics: truncate
		if length < minLen {
			length = minLen
		}
		if length > maxLen {
			length = maxLen
		}
		return float64(length)
	}
	return exptdLen
}
