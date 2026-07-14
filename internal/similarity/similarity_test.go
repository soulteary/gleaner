package similarity_test

import (
	"testing"

	"github.com/soulteary/gleaner/internal/similarity"
)

func TestCompareTwoStrings(t *testing.T) {
	if similarity.CompareTwoStrings("abc", "abc") != 1 {
		t.Fatal("identical should be 1")
	}
	if similarity.CompareTwoStrings("a", "b") != 0 {
		t.Fatal("short strings should be 0")
	}
	s := similarity.CompareTwoStrings("night", "nacht")
	if s <= 0 || s >= 1 {
		t.Fatalf("partial match score=%f", s)
	}
}

func TestCompareTokenStrings(t *testing.T) {
	if similarity.CompareTokenStrings([]string{"a", "b"}, []string{"a", "b"}) != 1 {
		t.Fatal("identical tokens should be 1")
	}
	if similarity.CompareTokenStrings([]string{"a"}, []string{"b"}) != 0 {
		t.Fatal("short tokens should be 0")
	}
	s := similarity.CompareTokenStrings([]string{"hello", "world"}, []string{"hello", "word"})
	if s <= 0 || s >= 1 {
		t.Fatalf("partial match score=%f", s)
	}
}
