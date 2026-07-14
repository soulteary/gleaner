package textrank_test

import (
	"testing"

	"github.com/soulteary/gleaner/internal/textrank"
)

func TestPageRankOriginalEmpty(t *testing.T) {
	if got := textrank.PageRankOriginal(nil, 0.85, 0.0001); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestPageRankOriginalSingleNode(t *testing.T) {
	graph := [][]float64{{0}}
	ranks := textrank.PageRankOriginal(graph, 0.85, 0.0001)
	if len(ranks) != 1 {
		t.Fatalf("len=%d", len(ranks))
	}
}

func TestPageRankOriginalDistanceIndex(t *testing.T) {
	// Mirrors node-summary: edge targets are Levenshtein distance values as indices.
	graph := [][]float64{
		{0, 3, 5},
		{3, 0, 2},
		{5, 2, 0},
	}
	ranks := textrank.PageRankOriginal(graph, 0.85, 0.0001)
	if len(ranks) != 3 {
		t.Fatalf("len=%d", len(ranks))
	}
	for _, r := range ranks {
		if r <= 0 {
			t.Fatalf("rank=%f", r)
		}
	}
}

func TestSimilarityGraphCommaJoin(t *testing.T) {
	// buildSimilarityGraph uses comma join; verify Distance on comma-joined tokens.
	a := []string{"第三方", "机构"}
	b := []string{"第三方", "机构"}
	sa := "第三方,机构"
	sb := "第三方,机构"
	if textrank.Distance(sa, sb) != 0 {
		t.Fatal("identical comma-joined should be 0")
	}
	_ = a
	_ = b
}
