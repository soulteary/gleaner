package extractor

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var lineBreakTags = []string{
	"address", "article", "aside", "audio", "blockquote", "canvas",
	"dd", "div", "dl", "fieldset", "figcaption", "figure", "footer", "form",
	"h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup", "hr", "noscript", "ol", "output", "p", "pre", "section",
	"table", "tr", "tfoot", "ul", "video", "img", "br",
}

var (
	commentRe      = regexp.MustCompile(`(?is)<!--[\s\S]*?-->`)
	scriptRe       = regexp.MustCompile(`(?is)<script[\s\S]*?</script>`)
	styleRe        = regexp.MustCompile(`(?is)<style[\s\S]*?</style>`)
	iframeRe       = regexp.MustCompile(`(?is)<iframe[\s\S]*?</iframe>`)
	whitespaceRe   = regexp.MustCompile(`[\n\t\r]+`)
	htmlTagRe      = regexp.MustCompile(`(?i)</?[^>]*?>`)
	entityRe       = regexp.MustCompile(`(?i)&[a-z]+;`)
	entityNumRe    = regexp.MustCompile(`&#(\d+);`)
	multiSpaceRe   = regexp.MustCompile(`\s{2,}`)
	blankLineRe    = regexp.MustCompile(`(?m)^\s+$`)
	bodyRe         = regexp.MustCompile(`(?is)^[\s\S]*(<body[\s\S]*?</body>)[\s\S]*$`)
	h1Re           = regexp.MustCompile(`(?i)<h1[^>]*?>([^<]+)</h1>`)
	titleRe        = regexp.MustCompile(`(?i)<title[^>]*?>([^<]+)</title>`)
	pubdateRe      = regexp.MustCompile(`([\d\-年月日]{8,15}\s?\d{1,2}[:：]\d{1,2})`)
	pubdateBlkRe   = regexp.MustCompile(`(\d{4}(?:\-|年)\d{1,2}(?:\-|月)\d{1,2}[\s日\d:]*)`)
	filterScript   = regexp.MustCompile(`(?is)<script[\s\S]*?/script>`)
	filterComment  = regexp.MustCompile(`(?is)<!--[\s\S]*?-->`)
	filterStyle    = regexp.MustCompile(`(?is)<style[\s\S]*?/style>`)
	filterScript2  = regexp.MustCompile(`(?i)<script[^>]+?>`)
	filterComment2 = regexp.MustCompile(`<!--[^>]+?>`)
	filterInput    = regexp.MustCompile(`(?i)<input[^>]+?>`)
	filterImg      = regexp.MustCompile(`(?i)<img.*?>`)
	filterLongA    = regexp.MustCompile(`(?i)<a[^>]*?>[^(</a>)]{10,}</a>`)
	filterNewline  = regexp.MustCompile(`[\n ]+`)
	filterManyNL   = regexp.MustCompile(`[\n]{4,}`)
)

// BlockGroup represents [start, end, rate].
type BlockGroup [3]float64

// FilterNoneContentTags removes non-content HTML elements.
func FilterNoneContentTags(html string) string {
	html = filterScript.ReplaceAllString(html, "")
	html = filterComment.ReplaceAllString(html, "")
	html = filterStyle.ReplaceAllString(html, "")
	html = filterScript2.ReplaceAllString(html, "")
	html = filterComment2.ReplaceAllString(html, "")
	html = filterInput.ReplaceAllString(html, "")
	html = filterImg.ReplaceAllString(html, "\n")
	html = filterLongA.ReplaceAllString(html, "")
	html = filterNewline.ReplaceAllString(html, "\n")
	html = filterManyNL.ReplaceAllString(html, "\n\n\n")
	return html
}

// GetBlocks splits HTML into text blocks by block-level tags.
func GetBlocks(html string) []string {
	html = commentRe.ReplaceAllString(html, "")
	html = scriptRe.ReplaceAllString(html, "")
	html = styleRe.ReplaceAllString(html, "")
	html = whitespaceRe.ReplaceAllString(html, "")
	splitter := strings.Join(lineBreakTags, "|")
	splitRe := regexp.MustCompile(`(?i)(<(?:` + splitter + `)[^>]*?>)`)
	html = splitRe.ReplaceAllString(html, "\n$1")
	html = htmlTagRe.ReplaceAllString(html, "")
	html = entityRe.ReplaceAllString(html, "")
	html = entityNumRe.ReplaceAllStringFunc(html, func(s string) string {
		m := entityNumRe.FindStringSubmatch(s)
		if len(m) < 2 {
			return s
		}
		var code int
		for _, c := range m[1] {
			code = code*10 + int(c-'0')
		}
		return string(rune(code))
	})
	paragraphs := strings.Split(html, "\n")
	out := make([]string, 0, len(paragraphs))
	for _, line := range paragraphs {
		line = multiSpaceRe.ReplaceAllString(line, " ")
		line = blankLineRe.ReplaceAllString(line, "")
		out = append(out, line)
	}
	return out
}

// blockLength returns character count matching JavaScript String.length for BMP text.
func blockLength(s string) int {
	return utf8.RuneCountInString(s)
}

// GetBlockScore computes block density scores with given depth.
func GetBlockScore(blocks []string, deep int) []int {
	if deep <= 0 || len(blocks) < deep {
		return nil
	}
	scores := make([]int, len(blocks)-deep+1)
	for i := 0; i <= len(blocks)-deep; i++ {
		score := 0
		for ix := i; ix < i+deep; ix++ {
			score += blockLength(blocks[ix])
		}
		scores[i] = score
	}
	return scores
}

// JudgeArticleBlock groups blocks by density and returns sorted groups [start, end, rate].
func JudgeArticleBlock(blockScore []int) []BlockGroup {
	var groups []BlockGroup
	seekStart := true
	startPoint := 0
	totalLength := 0
	maxBlockLength := 0

	for i, score := range blockScore {
		if score > maxBlockLength {
			maxBlockLength = score
		}
		if score > 0 {
			if seekStart {
				startPoint = i
				seekStart = false
			}
			totalLength += score
		} else if !seekStart {
			avg := float64(totalLength) / float64(i-startPoint)
			rate := float64(maxBlockLength*maxBlockLength) / avg
			groups = append(groups, BlockGroup{float64(startPoint), float64(i), rate})
			seekStart = true
			totalLength = 0
			maxBlockLength = 0
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i][2] > groups[j][2]
	})
	return groups
}

// GetTitle extracts article title from h1 or title tag.
func GetTitle(html string) string {
	h1Match := h1Re.FindStringSubmatch(html)
	titleMatch := titleRe.FindStringSubmatch(html)

	title := ""
	if len(h1Match) > 1 {
		title = h1Match[1]
	} else if len(titleMatch) > 1 {
		title = titleMatch[1]
	}

	if len(h1Match) > 1 && len(titleMatch) > 1 {
		tmch := strings.ReplaceAll(titleMatch[1], "\n", "")
		tmch = strings.ReplaceAll(tmch, "\r", "")
		hmch := strings.ReplaceAll(h1Match[1], "\n", "")
		hmch = strings.ReplaceAll(hmch, "\r", "")
		if tmch != "" && !strings.Contains(tmch, hmch[:min(10, len(hmch))]) {
			title = tmch
		}
	}
	title = strings.ReplaceAll(title, "\n", "")
	title = strings.ReplaceAll(title, "\r", "")
	return strings.TrimSpace(title)
}

// GetPubdate extracts publication date via regex from HTML.
func GetPubdate(html string) string {
	html = commentRe.ReplaceAllString(html, "")
	html = scriptRe.ReplaceAllString(html, "")
	html = styleRe.ReplaceAllString(html, "")
	if m := pubdateRe.FindStringSubmatch(html); len(m) > 1 {
		return m[1]
	}
	return ""
}

// GetPubdateBaseBlock finds pubdate near article start in blocks.
func GetPubdateBaseBlock(blocks []string, startPos int) string {
	for i := startPos; i >= 0; i-- {
		if m := pubdateBlkRe.FindStringSubmatch(blocks[i]); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// ExtractBody returns body content from full HTML.
func ExtractBody(html string) string {
	html = commentRe.ReplaceAllString(html, "")
	html = scriptRe.ReplaceAllString(html, "")
	html = styleRe.ReplaceAllString(html, "")
	html = iframeRe.ReplaceAllString(html, "")
	if m := bodyRe.FindStringSubmatch(html); len(m) > 1 {
		return m[1]
	}
	return html
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
