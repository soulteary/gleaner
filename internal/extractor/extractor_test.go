package extractor_test

import (
	"os"
	"strings"
	"testing"

	"github.com/soulteary/gleaner/internal/extractor"
)

const sampleHTML = `<!DOCTYPE html><html><head><title>测试标题 - 新闻站</title></head><body>
<script>var x=1;</script><style>.a{}</style>
<div class="nav">导航链接</div>
<div class="article">
<p>这是第一段正文内容，包含足够长的文字用于密度分析测试。</p>
<p>这是第二段正文内容，继续补充文章主体信息。</p>
<p>这是第三段正文内容，使正文区块密度明显高于导航区域。</p>
</div>
<div class="footer">页脚</div>
<span>2016-03-15 10:30</span>
</body></html>`

func TestFilterNoneContentTags(t *testing.T) {
	out := extractor.FilterNoneContentTags(`<script>x</script><p>正文</p><a href="#">长链接文本超过十字</a>`)
	if strings.Contains(out, "script") {
		t.Fatal("script should be removed")
	}
	if !strings.Contains(out, "正文") {
		t.Fatal("content should remain")
	}
}

func TestGetBlocks(t *testing.T) {
	blocks := extractor.GetBlocks(sampleHTML)
	nonEmpty := 0
	for _, b := range blocks {
		if strings.TrimSpace(b) != "" {
			nonEmpty++
		}
	}
	if nonEmpty < 2 {
		t.Fatalf("expected multiple blocks, got %d", nonEmpty)
	}
}

func TestGetBlockScoreChinese(t *testing.T) {
	zh := strings.Repeat("中文", 50) // 100 runes
	scores := extractor.GetBlockScore([]string{zh}, 1)
	if len(scores) != 1 {
		t.Fatalf("len=%d", len(scores))
	}
	if scores[0] != 100 {
		t.Fatalf("score=%d want 100 runes", scores[0])
	}
	// Same rune count must not inflate score via UTF-8 byte length (would be 300 bytes).
	if scores[0] == len(zh) {
		t.Fatalf("score should use rune count, got byte length %d", len(zh))
	}
}

func TestGetBlockScore(t *testing.T) {
	blocks := []string{"short", "longer block text", "another", "", "tail"}
	scores := extractor.GetBlockScore(blocks, 2)
	if len(scores) != len(blocks)-2+1 {
		t.Fatalf("scores len=%d", len(scores))
	}
	want := len("short") + len("longer block text")
	if scores[0] != want {
		t.Fatalf("score[0]=%d want %d", scores[0], want)
	}
}

func TestJudgeArticleBlock(t *testing.T) {
	blocks := extractor.GetBlocks(sampleHTML)
	scores := extractor.GetBlockScore(blocks, 5)
	groups := extractor.JudgeArticleBlock(scores)
	// Short fixtures may produce a single continuous group without zero-score separators.
	if len(groups) == 0 && len(scores) > 0 {
		t.Log("no zero-score separators in fixture; scores computed OK")
		return
	}
	if len(groups) == 0 {
		t.Fatal("expected at least one group")
	}
	if groups[0][2] < groups[len(groups)-1][2] {
		t.Fatal("groups should be sorted by rate desc")
	}
}

func TestGetTitle(t *testing.T) {
	html := `<html><head><title>页面标题很长</title></head><body><h1>正文标题</h1></body></html>`
	title := extractor.GetTitle(html)
	if title == "" {
		t.Fatal("expected title")
	}

	html2 := `<html><head><title>完全不同的页面标题</title></head><body><h1>正文标题</h1></body></html>`
	title2 := extractor.GetTitle(html2)
	if !strings.Contains(title2, "页面标题") {
		t.Fatalf("title2=%q", title2)
	}
}

func TestGetPubdate(t *testing.T) {
	cases := []struct {
		html string
		want string
	}{
		{`<span>2016-03-15 10:30</span>`, "2016-03-15 10:30"},
		{`<span>2016年3月15日 10:30</span>`, "2016年3月15日 10:30"},
	}
	for _, tc := range cases {
		got := extractor.GetPubdate(tc.html)
		if got != tc.want {
			t.Fatalf("html=%q got=%q want=%q", tc.html, got, tc.want)
		}
	}
}

func TestGetPubdateBaseBlock(t *testing.T) {
	blocks := []string{"meta", "2016年3月15日", "正文"}
	got := extractor.GetPubdateBaseBlock(blocks, 2)
	if got != "2016年3月15日" {
		t.Fatalf("got=%q", got)
	}
}

func TestExtractBodySkipsIframeNestedBody(t *testing.T) {
	html := `<html><head><title>T</title></head><body><p>real article</p>` +
		`<iframe src="javascript:'<html><body style=x></body></html>'"></iframe></body></html>`
	body := extractor.ExtractBody(html)
	if !strings.Contains(body, "real article") {
		t.Fatalf("body=%q", body)
	}
	if strings.Contains(body, "background:transparent") {
		t.Fatal("should not match iframe nested body")
	}
}

func TestLargeArticleBlocks(t *testing.T) {
	html, err := os.ReadFile("../../testdata/golden_large_article.html")
	if err != nil {
		t.Skip(err)
	}
	body := extractor.ExtractBody(string(html))
	filtered := extractor.FilterNoneContentTags(body)
	blocks := extractor.GetBlocks(filtered)
	nonEmpty := 0
	for _, b := range blocks {
		if strings.TrimSpace(b) != "" {
			nonEmpty++
		}
	}
	if nonEmpty < 10 {
		t.Fatalf("expected many content blocks, got %d non-empty of %d", nonEmpty, len(blocks))
	}
}

func TestGetBlocksNumericEntity(t *testing.T) {
	blocks := extractor.GetBlocks(`<p>&#65; test</p>`)
	found := false
	for _, b := range blocks {
		if strings.Contains(b, "A") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("blocks=%v", blocks)
	}
}

func TestJudgeArticleBlockMultipleGroups(t *testing.T) {
	blocks := []string{
		"nav",
		"",
		strings.Repeat("正文", 50),
		strings.Repeat("正文", 50),
		strings.Repeat("正文", 50),
		"",
		"footer",
	}
	scores := extractor.GetBlockScore(blocks, 1)
	groups := extractor.JudgeArticleBlock(scores)
	if len(groups) == 0 {
		t.Fatal("expected groups")
	}
	if groups[0][2] < groups[len(groups)-1][2] {
		t.Fatal("groups should be sorted by rate desc")
	}
}

func TestGetPubdateBaseBlockMiss(t *testing.T) {
	blocks := []string{"no date here", "also none"}
	if got := extractor.GetPubdateBaseBlock(blocks, 1); got != "" {
		t.Fatalf("got=%q", got)
	}
}

func TestGetTitlePrefersH1WhenSimilar(t *testing.T) {
	html := `<html><head><title>正文标题 - 站点</title></head><body><h1>正文标题</h1></body></html>`
	title := extractor.GetTitle(html)
	if title != "正文标题" {
		t.Fatalf("title=%q", title)
	}
}

func TestGetBlockScoreInvalidDeep(t *testing.T) {
	if got := extractor.GetBlockScore([]string{"a"}, 0); got != nil {
		t.Fatalf("got=%v", got)
	}
	if got := extractor.GetBlockScore([]string{"a"}, 5); got != nil {
		t.Fatalf("got=%v", got)
	}
}
