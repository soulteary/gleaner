//go:build cgo

package segment

import (
	"strings"

	"github.com/yanyiwu/gojieba"
)

type gojiebaSegmenter struct {
	jieba *gojieba.Jieba
}

func newGojieba(userDict string) (Segmenter, error) {
	// gojieba cannot safely load the AFINN one-word-per-line list (1555 entries);
	// it corrupts POS tagging and may crash DictTrie. nodejieba accepts that format,
	// but NN/summary quality depends on correct POS tags, so use the built-in dict.
	_ = userDict
	j := gojieba.NewJieba()
	return &gojiebaSegmenter{jieba: j}, nil
}

func (g *gojiebaSegmenter) Cut(text string) []string {
	return g.jieba.Cut(text, true)
}

func (g *gojiebaSegmenter) Tag(text string) []Token {
	tags := g.jieba.Tag(text)
	out := make([]Token, 0, len(tags))
	for _, t := range tags {
		parts := strings.SplitN(t, "/", 2)
		word := parts[0]
		pos := ""
		if len(parts) > 1 {
			pos = parts[1]
		}
		out = append(out, Token{Word: word, POS: pos})
	}
	return out
}

func (g *gojiebaSegmenter) Close() {
	if g.jieba != nil {
		g.jieba.Free()
		g.jieba = nil
	}
}
