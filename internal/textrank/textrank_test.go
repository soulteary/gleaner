package textrank_test

import (
	"math"
	"testing"

	"github.com/soulteary/gleaner/internal/textrank"
)

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"kitten", "sitting", 3},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"你好世界", "你好", 2},
		{"你好", "你好世界", 2},
		{"中文", "英文", 1},
	}
	for _, tc := range cases {
		got := textrank.Distance(tc.a, tc.b)
		if got != tc.want {
			t.Fatalf("%q vs %q: got %d want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestSimilarityGraph(t *testing.T) {
	sentences := [][]string{
		{"a", "b"},
		{"a", "c"},
		{"x", "y"},
	}
	graph := textrank.SimilarityGraph(sentences)
	if len(graph) != 3 || len(graph[0]) != 3 {
		t.Fatalf("graph dims=%dx%d", len(graph), len(graph[0]))
	}
	if graph[0][0] != 0 {
		t.Fatalf("self distance should be 0, got %v", graph[0][0])
	}
	if graph[0][1] >= graph[0][2] {
		t.Fatalf("ab closer than ac: %v vs %v", graph[0][1], graph[0][2])
	}
}

func TestSimilarityGraphEmptyTokens(t *testing.T) {
	graph := textrank.SimilarityGraph([][]string{{}, {"a"}})
	if len(graph) != 2 {
		t.Fatalf("len=%d", len(graph))
	}
	if graph[0][1] != float64(len("a")) {
		t.Fatalf("empty vs a distance=%v", graph[0][1])
	}
}

func TestPageRankEmpty(t *testing.T) {
	if textrank.PageRank(nil, 0.85, 0.0001) != nil {
		t.Fatal("nil graph should return nil")
	}
	if textrank.PageRank([][]float64{}, 0.85, 0.0001) != nil {
		t.Fatal("empty graph should return nil")
	}
}

func TestPageRankConverges(t *testing.T) {
	graph := [][]float64{
		{0, 1, 2},
		{1, 0, 1},
		{2, 1, 0},
	}
	ranks := textrank.PageRank(graph, 0.85, 0.0001)
	if len(ranks) != 3 {
		t.Fatalf("len=%d", len(ranks))
	}
	sum := 0.0
	for _, r := range ranks {
		sum += r
		if math.IsNaN(r) {
			t.Fatal("nan rank")
		}
	}
	if sum < 0.5 {
		t.Fatalf("sum=%f", sum)
	}
}

func TestPageRankSingleNode(t *testing.T) {
	graph := [][]float64{{0}}
	ranks := textrank.PageRank(graph, 0.85, 0.0001)
	if len(ranks) != 1 || ranks[0] <= 0 {
		t.Fatalf("ranks=%v", ranks)
	}
}

func TestPageRankOriginalConverges(t *testing.T) {
	graph := [][]float64{
		{0, 1, 2},
		{1, 0, 1},
		{2, 1, 0},
	}
	ranks := textrank.PageRankOriginal(graph, 0.85, 0.0001)
	if len(ranks) != 3 {
		t.Fatalf("len=%d", len(ranks))
	}
	sum := 0.0
	for _, r := range ranks {
		sum += r
		if math.IsNaN(r) {
			t.Fatal("nan rank")
		}
	}
	if sum < 0.5 {
		t.Fatalf("sum=%f", sum)
	}
}
