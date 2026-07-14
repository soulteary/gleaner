package segment

import (
	"strings"

	"github.com/go-ego/gse"
)

type gseSegmenter struct {
	seg gse.Segmenter
}

func newGSE(userDict string) (Segmenter, error) {
	var s gse.Segmenter
	if err := s.LoadDict(); err != nil {
		return nil, err
	}
	if userDict != "" {
		for _, line := range strings.Split(userDict, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) > 0 {
				_ = s.AddToken(parts[0], 100)
			}
		}
	}
	return &gseSegmenter{seg: s}, nil
}

func (g *gseSegmenter) Cut(text string) []string {
	return g.seg.Cut(text, true)
}

func (g *gseSegmenter) Tag(text string) []Token {
	tags := g.seg.Pos(text, true)
	out := make([]Token, 0, len(tags))
	for _, t := range tags {
		out = append(out, Token{Word: t.Text, POS: t.Pos})
	}
	return out
}

func (g *gseSegmenter) Close() {}
