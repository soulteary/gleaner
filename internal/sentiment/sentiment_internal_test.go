package sentiment

import (
	"encoding/json"
	"testing"

	"github.com/soulteary/gleaner/internal/segment"
	"github.com/soulteary/gleaner/internal/testutil"
)

type fakeSegmenter struct{}

func (fakeSegmenter) Cut(text string) []string { return []string{text} }
func (fakeSegmenter) Tag(text string) []segment.Token {
	return []segment.Token{{Word: text, POS: "n"}}
}
func (fakeSegmenter) Close() {}

func TestNewInvalidJSON(t *testing.T) {
	_, err := New(fakeSegmenter{}, []byte(`{bad`), []byte(`[]`))
	if err == nil {
		t.Fatal("expected afinn json error")
	}
	_, err = New(fakeSegmenter{}, []byte(`{}`), []byte(`{bad`))
	if err == nil {
		t.Fatal("expected protection json error")
	}
}

func TestTokenizeProtectionMerge(t *testing.T) {
	protection, _ := json.Marshal([]string{"一句", "话"})
	afinn, _ := json.Marshal(map[string]int{"话": 1})
	a, err := New(fakeSegmenter{}, afinn, protection)
	if err != nil {
		t.Fatal(err)
	}
	tokens := a.tokenize("这是一句话的例子")
	if len(tokens) == 0 {
		t.Fatal("expected tokens")
	}
}

func TestContainsHelper(t *testing.T) {
	if !contains([]string{"a", "b"}, "b") {
		t.Fatal("expected contains true")
	}
	if contains([]string{"a"}, "z") {
		t.Fatal("expected contains false")
	}
}

func TestAnalyzeWithInject(t *testing.T) {
	seg := testutil.NewSegmenter(t)
	a, err := New(seg, []byte(`{"开心":2}`), []byte(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	r := a.Analyze("今天很开心", map[string]int{"开心": 10})
	if r.Score <= 0 {
		t.Fatalf("score=%d", r.Score)
	}
}
