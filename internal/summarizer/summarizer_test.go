package summarizer_test

import (
	"strings"
	"testing"

	"github.com/soulteary/gleaner/internal/summarizer"
	"github.com/soulteary/gleaner/internal/testutil"
)

const articleContent = `一线楼市“调控风”来袭

去年以来深圳、上海房价上涨迅速，除了流动性相对宽裕、供应量下降及3·30新政等因素外，“首付贷”等操作也助涨市场。相关部门嗅到了其中被放大的购房杠杆风险。3月25日，上海发布《关于进一步完善本市住房市场体系和保障体系促进房地产市场平稳健康发展的若干意见》（下称“沪九条”），非沪籍限购“2改5”确定，上海二套房首付比例提高。当晚，深圳也发布新政，提高限购门槛和二套房首付比例。这标志着一线城市房地产调控的再次收紧。而除了一线城市的上海和深圳，重点二线城市中的武汉和南京也已出台楼市新政。

纵观一线城市的调控政策，核心其实就是限购和限贷两条。随着沪深两地明里暗里的调控，同为一线城市的北京和广州风声渐紧。从非户籍人口购房需缴纳的社保年限来看，北京和上海需要连续缴纳5年及以上，但上海还要求必须是“家庭购房”，即已婚，单身人士被排除在外。这一条，比北京更严。`

func TestSummarizeLength(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := summarizer.New(seg, m, false, nil)
	out := s.Summarize(articleContent, 200, "一线楼市“调控风”来袭")
	if out == "" {
		t.Fatal("empty summary")
	}
	if len([]rune(out)) > 400 {
		t.Fatalf("summary too long: %d", len([]rune(out)))
	}
}

func TestSummarizeDeterministic(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := summarizer.New(seg, m, false, nil)
	a := s.Summarize(articleContent, 120, "一线楼市“调控风”来袭")
	b := s.Summarize(articleContent, 120, "一线楼市“调控风”来袭")
	if a != b {
		t.Fatalf("non-deterministic summary")
	}
}

func TestShingleMode(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	plain := summarizer.New(seg, m, false, nil).Summarize(articleContent, 120, "")
	shingle := summarizer.New(seg, m, true, nil).Summarize(articleContent, 120, "")
	if plain == "" || shingle == "" {
		t.Fatal("empty summary")
	}
}

func TestResolveExpectedLengthRatio(t *testing.T) {
	blocks := []string{"abc", "def"}
	got := summarizer.ResolveExpectedLength(blocks, 0.04, 100, 300)
	if got != 100 {
		t.Fatalf("got=%v want min 100", got)
	}
	blocks = []string{strings.Repeat("文", 1000)}
	got = summarizer.ResolveExpectedLength(blocks, 0.04, 100, 300)
	if got < 100 || got > 300 {
		t.Fatalf("got=%v", got)
	}
}
