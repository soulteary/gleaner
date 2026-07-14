package gleaner_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/soulteary/gleaner"
)

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")
	return s
}

type goldenExpected struct {
	Title           string   `json:"title"`
	SummaryKeywords []string `json:"summary_keywords"`
	ContentKeywords []string `json:"content_keywords"`
	SentimentMin    int      `json:"sentiment_min"`
	ContentRunesMin int      `json:"content_runes_min"`
	ContentRunesMax int      `json:"content_runes_max"`
}

func loadGoldenExpected(t *testing.T) goldenExpected {
	t.Helper()
	data, err := os.ReadFile("testdata/golden_expected.json")
	if err != nil {
		t.Skip("testdata/golden_expected.json not present")
	}
	var exp goldenExpected
	if err := json.Unmarshal(data, &exp); err != nil {
		t.Fatal(err)
	}
	return exp
}

func TestGoldenArticleSnapshot(t *testing.T) {
	html, err := os.ReadFile("testdata/golden_article.html")
	if err != nil {
		t.Skip("testdata/golden_article.html not present")
	}
	exp := loadGoldenExpected(t)

	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	shingle := false
	article, err := client.GetArticleSync(string(html), 200, &shingle)
	if err != nil {
		t.Fatal(err)
	}

	if article.Title != exp.Title {
		t.Fatalf("title=%q want %q", article.Title, exp.Title)
	}

	normContent := normalizeText(article.Content)
	runes := utf8.RuneCountInString(normContent)
	if runes < exp.ContentRunesMin || runes > exp.ContentRunesMax {
		t.Fatalf("content runes=%d want [%d,%d]", runes, exp.ContentRunesMin, exp.ContentRunesMax)
	}
	for _, kw := range exp.ContentKeywords {
		if !strings.Contains(normContent, normalizeText(kw)) {
			t.Fatalf("content missing keyword %q", kw)
		}
	}

	if article.Summary == "" {
		t.Fatal("empty summary")
	}
	for _, kw := range exp.SummaryKeywords {
		if !strings.Contains(article.Summary, kw) {
			t.Fatalf("summary missing keyword %q in %q", kw, article.Summary)
		}
	}

	if int(article.Sentiment) < exp.SentimentMin {
		t.Fatalf("sentiment=%v want >= %d", article.Sentiment, exp.SentimentMin)
	}
}

func TestGoldenSummarizeDeterministic(t *testing.T) {
	html, err := os.ReadFile("testdata/golden_article.html")
	if err != nil {
		t.Skip("testdata/golden_article.html not present")
	}
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	shingle := false
	a := client.Summarize(string(html), 200, &shingle, "星河一号AI芯片")
	b := client.Summarize(string(html), 200, &shingle, "星河一号AI芯片")
	if a != b {
		t.Fatalf("non-deterministic summarize")
	}
}

func TestGoldenSummarizeRatioLength(t *testing.T) {
	html, err := os.ReadFile("testdata/golden_article.html")
	if err != nil {
		t.Skip("testdata/golden_article.html not present")
	}
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	summary := client.Summarize(string(html), 0.04, nil, "")
	if summary == "" {
		t.Fatal("empty ratio summary")
	}
	// DefaultOptions: MinSummaryLen=150, MaxSummaryLen=350。修正后的选句循环按字符
	// 累加长度并"先纳入再判断"，摘要会贴近期望长度而不再偏短，因此下限收紧到 120。
	if n := utf8.RuneCountInString(summary); n < 120 || n > 350 {
		t.Fatalf("ratio summary runes=%d", n)
	}
}
