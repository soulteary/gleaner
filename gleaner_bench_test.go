package gleaner_test

import (
	"os"
	"testing"

	"github.com/soulteary/gleaner"
)

func BenchmarkSummarize(b *testing.B) {
	html, err := os.ReadFile("testdata/golden_large_article.html")
	if err != nil {
		b.Skip(err)
	}
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	shingle := true
	content := string(html)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Summarize(content, 200, &shingle, "星河一号AI芯片")
	}
}

func BenchmarkGetArticleSync(b *testing.B) {
	html, err := os.ReadFile("testdata/golden_large_article.html")
	if err != nil {
		b.Skip(err)
	}
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	shingle := true
	content := string(html)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetArticleSync(content, 200, &shingle)
	}
}
