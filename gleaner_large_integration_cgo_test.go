//go:build cgo

package gleaner_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/soulteary/gleaner"
)

func TestGoldenLargeArticleSnapshot(t *testing.T) {
	html, err := os.ReadFile("testdata/golden_large_article.html")
	if err != nil {
		t.Skip("testdata/golden_large_article.html not present")
	}
	exp := loadLargeGoldenExpected(t)

	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	shingle := true // matches arex-master/test/test-summarize.js
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

type largeGoldenExpected struct {
	Title           string   `json:"title"`
	SummaryKeywords []string `json:"summary_keywords"`
	ContentKeywords []string `json:"content_keywords"`
	SentimentMin    int      `json:"sentiment_min"`
	ContentRunesMin int      `json:"content_runes_min"`
	ContentRunesMax int      `json:"content_runes_max"`
}

func loadLargeGoldenExpected(t *testing.T) largeGoldenExpected {
	t.Helper()
	data, err := os.ReadFile("testdata/golden_large_expected.json")
	if err != nil {
		t.Skip("testdata/golden_large_expected.json not present")
	}
	var exp largeGoldenExpected
	if err := json.Unmarshal(data, &exp); err != nil {
		t.Fatal(err)
	}
	return exp
}
