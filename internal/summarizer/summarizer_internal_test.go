package summarizer

import (
	"strings"
	"testing"

	"github.com/soulteary/gleaner/internal/testutil"
)

func TestIsValidateSentenceASCIIHeavy(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	m := testutil.NewModel(t)
	s := New(seg, m, false, nil)
	if s.isValidateSentence([]string{"hello", "world", "test", "case"}) {
		t.Fatal("ASCII-heavy sentence should be invalid")
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
