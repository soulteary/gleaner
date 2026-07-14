package summarizer

import (
	"math"
	"strings"
	"testing"

	"github.com/soulteary/gleaner/internal/testutil"
	"github.com/soulteary/gleaner/internal/textrank"
)

func TestIsValidateSentenceASCIIHeavy(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := New(seg, m, false, nil)
	if s.isValidateSentence([]string{"hello", "world", "test", "case"}) {
		t.Fatal("ASCII-heavy sentence should be invalid")
	}
}

// TestIsValidateSentenceMixedCJK 验证含英文型号/数字的中文技术句不再被 ASCII 规则误删。
func TestIsValidateSentenceMixedCJK(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := New(seg, m, false, nil)
	cases := [][]string{
		{"星河", "一号", "AI", "芯片", "算力", "提升", "3", "倍", "。"},
		{"搭载", "M2", "处理器", "的", "设备", "性能", "更强", "。"},
	}
	for _, c := range cases {
		if !s.isValidateSentence(c) {
			t.Fatalf("Chinese sentence with ASCII tokens should be valid: %v", c)
		}
	}
}

func TestIsValidateSentenceFilter(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := New(seg, m, false, []string{"广告"})
	if s.isValidateSentence([]string{"这是", "广告", "内容", "。"}) {
		t.Fatal("filtered sentence should be invalid")
	}
}

func TestIsValidateSentenceEmpty(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := New(seg, m, false, nil)
	if s.isValidateSentence(nil) {
		t.Fatal("empty should be invalid")
	}
}

func TestResolveExpectedLengthPassthrough(t *testing.T) {
	got := ResolveExpectedLength([]string{"abc"}, 250, 100, 300)
	if got != 250 {
		t.Fatalf("got=%v", got)
	}
}

func TestResolveExpectedLengthMaxCap(t *testing.T) {
	blocks := []string{strings.Repeat("文", 10000)}
	got := ResolveExpectedLength(blocks, 0.5, 100, 300)
	if got != 300 {
		t.Fatalf("got=%v want max 300", got)
	}
}

// TestSimilarityGraphIsSimilarity 验证 buildSimilarityGraph 产出的是相似度（越相似值越大）
// 而非距离：完全不同的句子相似度应低于内容相近的句子，且自环为 0。
func TestSimilarityGraphIsSimilarity(t *testing.T) {
	sentences := [][]string{
		{"人工", "智能", "芯片"},
		{"人工", "智能", "算法"},
		{"天气", "很好"},
	}
	g := buildSimilarityGraph(sentences)
	if g[0][0] != 0 {
		t.Fatalf("self-loop should be 0, got %v", g[0][0])
	}
	// 句 0 与句 1 内容相近，应比句 0 与句 2 更相似（相似度更大）。
	if g[0][1] <= g[0][2] {
		t.Fatalf("expected sim(0,1)=%v > sim(0,2)=%v", g[0][1], g[0][2])
	}
	// 相似度取值应落在 [0,1]。
	for i := range g {
		for j := range g[i] {
			if g[i][j] < 0 || g[i][j] > 1 {
				t.Fatalf("sim(%d,%d)=%v out of [0,1]", i, j, g[i][j])
			}
		}
	}
}

// TestTextRankHasDiscrimination 验证修正后的 相似度图 + 标准 PageRank
// 让 TextRank 分数具有区分度（方差 > 0），而非退化成常数。
func TestTextRankHasDiscrimination(t *testing.T) {
	sentences := [][]string{
		{"人工", "智能", "芯片", "算力", "提升"},
		{"人工", "智能", "芯片", "功耗", "降低"},
		{"大", "模型", "训练", "推理", "生态"},
		{"国产", "芯片", "算力", "能效", "突破"},
		{"今天", "天气", "很好", "适合", "出行"},
	}
	graph := buildSimilarityGraph(sentences)
	rank := textrank.PageRank(graph, 0.85, 0.0001)
	if len(rank) != len(sentences) {
		t.Fatalf("rank len=%d want %d", len(rank), len(sentences))
	}
	mean := 0.0
	for _, r := range rank {
		if math.IsNaN(r) {
			t.Fatal("NaN rank")
		}
		mean += r
	}
	mean /= float64(len(rank))
	variance := 0.0
	for _, r := range rank {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(rank))
	if variance <= 1e-9 {
		t.Fatalf("TextRank scores show no discrimination, variance=%g", variance)
	}
}
