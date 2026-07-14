package summarizer

import (
	"math"
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
	lens := make([]int, len(sentences))
	for i, sent := range sentences {
		lens[i] = len([]rune(strings.Join(sent, "")))
	}

	// titleRepeat 标记与标题高度相似的"标题复述句"，这类句子会让摘要首句复述标题，
	// 应在候选阶段直接判为无效，不参与选句。阈值 0.8（修复后的中文 Dice 已生效）。
	const titleRepeatThreshold = 0.8
	titleRepeat := make([]bool, len(sentences))

	for i := range sentences {
		originSentence := strings.Join(sentences[i], "")
		spacesSentence := strings.Join(sentences[i], " ")
		titleSim := 0.0
		if spacesTitle != "" {
			titleSim = similarity.CompareTwoStrings(spacesTitle, spacesSentence)
			if titleSim >= titleRepeatThreshold {
				titleRepeat[i] = true
			}
		}
		modelScore := s.model.Rank(s.seg, originSentence)
		// 去除长句奖励，改为对过短句做惩罚：短于 ~6 个 rune 的句子信息量低，
		// 线性惩罚到 0；达到阈值后不再影响得分（不奖励长句）。
		const shortLenFloor = 6
		lenPenalty := 1.0
		if lens[i] < shortLenFloor {
			lenPenalty = float64(lens[i]) / float64(shortLenFloor)
		}
		// 重新平衡权重：提高 textRank 与（修复后的）titleSim，下调 modelScore，
		// 长度只作为轻微惩罚项而非长句偏置。
		finalScore := textRank[i]*0.55 + titleSim*0.25 + modelScore*0.15 + lenPenalty*0.05
		copiedRank[i] = ranked{score: finalScore, idx: i}
	}

	// 稳定排序：分数相等时按原始下标升序，保证确定性。
	sort.SliceStable(copiedRank, func(i, j int) bool {
		if copiedRank[i].score != copiedRank[j].score {
			return copiedRank[i].score > copiedRank[j].score
		}
		return copiedRank[i].idx < copiedRank[j].idx
	})

	// MMR 去冗余选句：每步在候选中选出
	//   argmax [ λ·score(i) − (1−λ)·max_{j∈selected} sim(i,j) ]
	// 的句子，sim 复用词汇重叠相似度图。λ=0.7 兼顾相关性与多样性。
	// 保持"先纳入再判断长度"的既有语义：达到 expectedLength 后停止。
	const lambda = 0.7
	baseScore := make(map[int]float64, len(copiedRank))
	remaining := make([]int, 0, len(copiedRank))
	for _, r := range copiedRank {
		if !s.isValidateSentence(sentences[r.idx]) || titleRepeat[r.idx] {
			continue
		}
		baseScore[r.idx] = r.score
		remaining = append(remaining, r.idx)
	}

	var selectIndex []int
	summarySize := 0
	for len(remaining) > 0 {
		bestPos := -1
		bestMMR := math.Inf(-1)
		for pos, idx := range remaining {
			redundancy := 0.0
			for _, sel := range selectIndex {
				if graph[idx][sel] > redundancy {
					redundancy = graph[idx][sel]
				}
			}
			mmr := lambda*baseScore[idx] - (1-lambda)*redundancy
			// 稳定选择：MMR 相等时优先原始下标较小者（remaining 已按分数+下标稳定排序）。
			if mmr > bestMMR {
				bestMMR = mmr
				bestPos = pos
			}
		}
		chosen := remaining[bestPos]
		remaining = append(remaining[:bestPos], remaining[bestPos+1:]...)

		selectIndex = append(selectIndex, chosen)
		summarySize += lens[chosen]
		if float64(summarySize) >= expectedLength {
			break
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

// stopwords 是相似度计算时应忽略的高频虚词/标点 token 集合。它们对句子的
// 主题贡献极低，若计入词汇重叠会让"的/了/是"等词主导 sim，掩盖真正的主题词重叠。
var stopwords = map[string]struct{}{
	"的": {}, "了": {}, "是": {}, "在": {}, "和": {}, "与": {}, "及": {},
	"也": {}, "都": {}, "而": {}, "并": {}, "或": {}, "但": {}, "却": {},
	"就": {}, "还": {}, "又": {}, "很": {}, "更": {}, "最": {}, "被": {},
	"把": {}, "对": {}, "为": {}, "以": {}, "于": {}, "从": {}, "到": {},
	"向": {}, "由": {}, "让": {}, "使": {}, "这": {}, "那": {}, "其": {},
	"该": {}, "此": {}, "之": {}, "着": {}, "过": {}, "地": {}, "得": {},
	"等": {}, "中": {}, "上": {}, "下": {}, "里": {}, "内": {}, "外": {},
	"有": {}, "无": {}, "个": {}, "些": {}, "会": {}, "能": {}, "要": {},
	"将": {}, "已": {}, "即": {}, "则": {}, "至": {}, "所": {},
	"。": {}, "，": {}, "、": {}, "：": {}, "；": {}, "？": {}, "！": {},
	"“": {}, "”": {}, "‘": {}, "’": {}, "（": {}, "）": {}, "《": {}, "》": {},
	"…": {}, "—": {}, ".": {}, ",": {}, ":": {}, ";": {}, "?": {}, "!": {},
}

// isStopToken 判断一个 token 是否为停用词或纯标点/空白，应从相似度词集中剔除。
func isStopToken(tok string) bool {
	if tok == "" {
		return true
	}
	if _, ok := stopwords[tok]; ok {
		return true
	}
	for _, r := range tok {
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) && !unicode.IsSymbol(r) {
			return false
		}
	}
	// 全为标点/空白/符号的 token 视为停用。
	return true
}

// sentenceWordFreq 构建句子的实词词频（已剔除停用词/标点），用于词汇重叠相似度。
func sentenceWordFreq(sentence []string) map[string]int {
	freq := make(map[string]int, len(sentence))
	for _, tok := range sentence {
		if isStopToken(tok) {
			continue
		}
		freq[tok]++
	}
	return freq
}

// overlapSimilarity 计算两句的归一化词汇重叠，采用 TextRank 经典公式：
//
//	sim(i,j) = |common| / (log(len_i) + log(len_j))
//
// 其中 len 为剔除停用词后的实词数量，common 为按词频取 min 累计的共现词数。
// 当分母不可用（句子过短导致 log 退化）时退回 overlap/(len_i+len_j) 作为保护，
// 结果统一裁剪到 [0,1]。空词集或无重叠返回 0。
func overlapSimilarity(fi, fj map[string]int, li, lj int) float64 {
	if li == 0 || lj == 0 {
		return 0
	}
	common := 0
	// 遍历较小的词表以减少查找次数。
	small, large := fi, fj
	if len(fj) < len(fi) {
		small, large = fj, fi
	}
	for w, c := range small {
		if c2, ok := large[w]; ok {
			if c2 < c {
				common += c2
			} else {
				common += c
			}
		}
	}
	if common == 0 {
		return 0
	}
	denom := math.Log(float64(li)) + math.Log(float64(lj))
	if denom <= 0 {
		// 句子过短（len<=1 使 log<=0）时的退化保护。
		denom = float64(li + lj)
	}
	sim := float64(common) / denom
	if sim < 0 {
		sim = 0
	}
	if sim > 1 {
		sim = 1
	}
	return sim
}

// buildSimilarityGraph 产出用于 TextRank 的归一化相似度矩阵。相似度基于剔除停用词后的
// 词汇重叠（越相似值越大），比编辑距离更能反映中文句子间的主题重合。自环 (i==j) 置 0
// 避免主导图；取值裁剪到 [0,1]。矩阵按 PageRank 期望的语义（值越大=越相似）返回，
// PageRank 接口与数值语义均不改动。
func buildSimilarityGraph(sentences [][]string) [][]float64 {
	n := len(sentences)
	graph := make([][]float64, n)
	freqs := make([]map[string]int, n)
	lens := make([]int, n)
	for i := 0; i < n; i++ {
		freqs[i] = sentenceWordFreq(sentences[i])
		total := 0
		for _, c := range freqs[i] {
			total += c
		}
		lens[i] = total
	}
	for i := 0; i < n; i++ {
		graph[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			graph[i][j] = overlapSimilarity(freqs[i], freqs[j], lens[i], lens[j])
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
