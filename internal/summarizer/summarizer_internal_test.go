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

// TestSimilarityGraphIgnoresStopwords 验证词汇重叠相似度会剔除停用词/标点：
// 仅共享停用词的两句相似度应为 0，共享实词的两句应为正。
func TestSimilarityGraphIgnoresStopwords(t *testing.T) {
	sentences := [][]string{
		{"人工", "智能", "的", "芯片", "。"},
		{"天气", "的", "很好", "。"},
		{"人工", "智能", "的", "算法", "。"},
	}
	g := buildSimilarityGraph(sentences)
	// 句 0 与句 1 只共享停用词"的""。"，实词无重叠，相似度应为 0。
	if g[0][1] != 0 {
		t.Fatalf("stopword-only overlap should be 0, got %v", g[0][1])
	}
	// 句 0 与句 2 共享实词"人工""智能"，相似度应为正。
	if g[0][2] <= 0 {
		t.Fatalf("real-word overlap should be > 0, got %v", g[0][2])
	}
}

// TestSummarizeDropsTitleRepeat 验证与标题几乎相同的复述句不会成为摘要首句。
func TestSummarizeDropsTitleRepeat(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := New(seg, m, false, nil)
	title := "国产人工智能芯片实现算力突破"
	content := "国产人工智能芯片实现算力突破。" +
		"研究团队表示新架构在能效方面取得显著进展并降低了整体功耗。" +
		"该芯片已进入量产阶段并计划在多家数据中心部署应用。" +
		"业内分析认为这将推动大模型训练与推理生态的发展。"
	out := s.Summarize(content, 120, title)
	if out == "" {
		t.Fatal("empty summary")
	}
	// 摘要不应以与标题高度重合的复述句开头。
	first := strings.SplitN(out, "...", 2)[0]
	if similarityToTitle := diceLike(title, first); similarityToTitle >= 0.8 {
		t.Fatalf("summary starts with title restatement: %q", first)
	}
}

// diceLike 是测试内联的 rune 级 bigram Dice，用于断言首句不复述标题，
// 避免直接依赖 similarity 包的私有行为。
func diceLike(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	if a == b {
		return 1
	}
	if len(ra) < 2 || len(rb) < 2 {
		return 0
	}
	bg := map[string]int{}
	for i := 0; i < len(ra)-1; i++ {
		bg[string(ra[i:i+2])]++
	}
	inter := 0
	for i := 0; i < len(rb)-1; i++ {
		k := string(rb[i : i+2])
		if bg[k] > 0 {
			bg[k]--
			inter++
		}
	}
	return 2 * float64(inter) / float64(len(ra)+len(rb)-2)
}

// TestSummarizeMMRAvoidsRedundancy 验证 MMR 选句不会选入内容近乎重复的句子：
// 构造两句几乎相同、另有若干不同主题句，摘要中不应同时包含那对重复句。
func TestSummarizeMMRAvoidsRedundancy(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := New(seg, m, false, nil)
	// 两句几乎相同（仅末尾差异），其余为不同主题。
	content := "人工智能芯片算力大幅提升带动产业升级。" +
		"人工智能芯片算力大幅提升带动产业发展。" +
		"新能源汽车销量持续增长市场前景广阔。" +
		"生物医药研发投入加大创新成果不断涌现。" +
		"航空航天技术取得重要进展探索深空。"
	out := s.Summarize(content, 60, "")
	parts := strings.Split(out, "...")
	dupCount := 0
	for _, p := range parts {
		if strings.Contains(p, "人工智能芯片算力大幅提升带动产业") {
			dupCount++
		}
	}
	if dupCount > 1 {
		t.Fatalf("MMR selected redundant near-duplicate sentences: %q", out)
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
