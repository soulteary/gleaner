package nnmodel_test

import (
	"math"
	"testing"

	"github.com/soulteary/gleaner/internal/data"
	"github.com/soulteary/gleaner/internal/nnmodel"
	"github.com/soulteary/gleaner/internal/segment"
	"github.com/soulteary/gleaner/internal/testutil"
)

func TestLoadAndValidate(t *testing.T) {
	m, err := nnmodel.Load(data.SummaryJSON)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateFeature(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	feat := nnmodel.GenerateFeature(seg, "第三方机构网贷之家公布数据")
	if len(feat) != nnmodel.FeatureSize {
		t.Fatalf("len=%d", len(feat))
	}
	nonZero := 0
	for _, v := range feat {
		if v != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatal("expected some non-zero POS features")
	}
}

func TestRankRange(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	sentence := "第三方机构网贷之家公布数据：截至2015年底，全年网贷成交量达到了9823.04亿元，比2014年全年成交量增长了288.57%。"
	score := m.Rank(seg, sentence)
	if score < 0 || score > 1 || math.IsNaN(score) {
		t.Fatalf("score=%f", score)
	}
}

func TestRankReferenceSentence(t *testing.T) {
	seg, err := segment.New(segment.BackendGojieba, data.UserWordsTXT)
	if err != nil {
		t.Skip("gojieba reference score requires CGO")
	}
	defer seg.Close()
	m := testutil.NewModel(t)
	sentence := "第三方机构网贷之家公布数据：截至2015年底，全年网贷成交量达到了9823.04亿元，比2014年全年成交量增长了288.57%。"
	score := m.Rank(seg, sentence)
	// Node isummary.rank reference (nodejieba without user dict, brain.js model).
	const want = 0.12022049725055695
	if math.Abs(score-want) > 1e-4 {
		t.Fatalf("score=%g want≈%g (tolerance 1e-4)", score, want)
	}
}

func TestRankDeterministic(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	sentence := "一线楼市调控风来袭，上海深圳房价上涨迅速。"
	a := m.Rank(seg, sentence)
	b := m.Rank(seg, sentence)
	if math.Abs(a-b) > 1e-9 {
		t.Fatalf("non-deterministic: %g vs %g", a, b)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	_, err := nnmodel.Load([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestValidateWrongSizes(t *testing.T) {
	m := &nnmodel.Model{Sizes: []int{1, 2, 3}}
	if err := m.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDefaultModel(t *testing.T) {
	m, err := nnmodel.DefaultModel(data.SummaryJSON)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	m2, err := nnmodel.DefaultModel(data.SummaryJSON)
	if err != nil {
		t.Fatal(err)
	}
	if m != m2 {
		t.Fatal("DefaultModel should return singleton")
	}
}
