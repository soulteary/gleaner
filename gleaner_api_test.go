package gleaner_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/soulteary/gleaner"
)

func init() {
	_ = os.Setenv("GLEANER_BLOCK_PRIVATE_URLS", "false")
}

func TestDefaultClient(t *testing.T) {
	c1, err := gleaner.DefaultClient()
	if err != nil {
		t.Fatal(err)
	}
	c2, err := gleaner.DefaultClient()
	if err != nil {
		t.Fatal(err)
	}
	if c1 != c2 {
		t.Fatal("DefaultClient should return singleton")
	}
}

func TestPackageGetArticleSync(t *testing.T) {
	article, err := gleaner.GetArticleSync(goldenHTML, 200, nil)
	if err != nil {
		t.Fatal(err)
	}
	if article.Content == "" || article.Summary == "" {
		t.Fatalf("article=%+v", article)
	}
}

func TestPackageSummarize(t *testing.T) {
	summary, err := gleaner.Summarize(goldenHTML, 120, nil, "星河一号AI芯片")
	if err != nil {
		t.Fatal(err)
	}
	if summary == "" {
		t.Fatal("empty summary")
	}
}

func TestPackageSentiment(t *testing.T) {
	r, err := gleaner.Sentiment("非常好看，开心", nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Score == 0 && len(r.Tokens) == 0 {
		t.Fatal("expected sentiment result")
	}
}

func TestGetArticleWithContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(goldenHTML))
	}))
	defer srv.Close()

	opts := gleaner.DefaultOptions()
	opts.BlockPrivateURLs = false
	client, err := gleaner.NewClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	article, err := client.GetArticleWithContext(t.Context(), srv.URL, gleaner.DefaultExtractOptions(200, nil))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(article.Content, "星河一号") {
		t.Fatalf("content=%q", article.Content[:min(80, len(article.Content))])
	}
}

func TestPackageGetArticle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(goldenHTML))
	}))
	defer srv.Close()

	t.Setenv("GLEANER_BLOCK_PRIVATE_URLS", "false")
	article, err := gleaner.GetArticle(srv.URL, 200, nil)
	if err != nil {
		t.Fatal(err)
	}
	if article.Title == "" {
		t.Fatal("empty title")
	}
}

func TestGetArticleSyncWithOptionsFilter(t *testing.T) {
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	opts := gleaner.ExtractOptions{
		ExpectedLen:   200,
		MinSummaryLen: 100,
		MaxSummaryLen: 300,
		Filter:        []string{"专题"},
	}
	article, err := client.GetArticleSyncWithOptions(goldenHTML, opts)
	if err != nil {
		t.Fatal(err)
	}
	if article.Summary == "" {
		t.Fatal("empty summary")
	}
}

func TestClientGetArticleSyncEmptyGroups(t *testing.T) {
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	html := `<html><head><title>短</title></head><body><p>单段正文。</p></body></html>`
	article, err := client.GetArticleSync(html, 50, nil)
	if err != nil {
		t.Fatal(err)
	}
	if article.Content == "" {
		t.Fatal("expected content from fallback blocks")
	}
}
