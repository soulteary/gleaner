package sentiment_test

import (
	"testing"

	"github.com/soulteary/gleaner/internal/data"
	"github.com/soulteary/gleaner/internal/sentiment"
	"github.com/soulteary/gleaner/internal/testutil"
)

func TestPositiveSentiment(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	a, err := sentiment.New(seg, data.AFINNJSON, data.DUProtectionJSON)
	if err != nil {
		t.Fatal(err)
	}
	r := a.Analyze("建国大业非常好看，今天过得太开心了！", nil)
	if r.Score <= 0 {
		t.Fatalf("expected positive score, got %d", r.Score)
	}
}

func TestNegativeSentiment(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	a, err := sentiment.New(seg, data.AFINNJSON, data.DUProtectionJSON)
	if err != nil {
		t.Fatal(err)
	}
	r := a.Analyze("一文不值，一拖再拖", nil)
	if r.Score >= 0 {
		t.Fatalf("expected negative score, got %d", r.Score)
	}
}

func TestInjectLexicon(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	a, err := sentiment.New(seg, data.AFINNJSON, data.DUProtectionJSON)
	if err != nil {
		t.Fatal(err)
	}
	r := a.Analyze("开心", map[string]int{"开心": -10})
	if r.Score >= 0 {
		t.Fatalf("expected negative from inject, got %d", r.Score)
	}
}

func TestScore(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	a, err := sentiment.New(seg, data.AFINNJSON, data.DUProtectionJSON)
	if err != nil {
		t.Fatal(err)
	}
	score := a.Score("建国大业非常好看，今天过得太开心了！")
	if score <= 0 {
		t.Fatalf("expected positive score, got %d", score)
	}
}

func TestMergeInject(t *testing.T) {
	base := map[string]int{"好": 3}
	if merged := sentiment.MergeInject(base, nil); merged == nil || merged["好"] != 3 {
		t.Fatal("nil inject should return equivalent base")
	}
	merged := sentiment.MergeInject(base, map[string]int{" 好 ": 5, "坏": -2})
	if merged["好"] != 5 || merged["坏"] != -2 {
		t.Fatalf("merged=%v", merged)
	}
}
