package extractor_test

import (
	"os"
	"testing"

	"github.com/soulteary/gleaner/internal/extractor"
)

func BenchmarkGetBlocks(b *testing.B) {
	html, err := os.ReadFile("../../testdata/golden_large_article.html")
	if err != nil {
		b.Skip(err)
	}
	body := extractor.ExtractBody(string(html))
	body = extractor.FilterNoneContentTags(body)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.GetBlocks(body)
	}
}

func BenchmarkGetBlockScore(b *testing.B) {
	html, err := os.ReadFile("../../testdata/golden_large_article.html")
	if err != nil {
		b.Skip(err)
	}
	body := extractor.ExtractBody(string(html))
	body = extractor.FilterNoneContentTags(body)
	blocks := extractor.GetBlocks(body)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.GetBlockScore(blocks, 5)
	}
}
