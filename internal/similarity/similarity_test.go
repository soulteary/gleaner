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

// TestCompareTwoStringsChinese 验证 rune 级 bigram 使中文相似度生效：
// 相同中文串为 1，重叠子串给出正分，完全不同为 0，且不再受 UTF-8 字节错位干扰。
func TestCompareTwoStringsChinese(t *testing.T) {
	if similarity.CompareTwoStrings("人工智能芯片", "人工智能芯片") != 1 {
		t.Fatal("identical Chinese should be 1")
	}
	overlap := similarity.CompareTwoStrings("人工智能芯片发布", "人工智能算法研究")
	if overlap <= 0 || overlap >= 1 {
		t.Fatalf("overlapping Chinese should be in (0,1), got %f", overlap)
	}
	disjoint := similarity.CompareTwoStrings("今天天气很好", "股市大幅上涨")
	if disjoint != 0 {
		t.Fatalf("disjoint Chinese should be 0, got %f", disjoint)
	}
	// 共享前缀更多的句子相似度应更高（编辑距离/字节切分难以稳定区分）。
	more := similarity.CompareTwoStrings("国产芯片算力突破", "国产芯片能效突破")
	less := similarity.CompareTwoStrings("国产芯片算力突破", "今天天气非常好")
	if more <= less {
		t.Fatalf("expected more=%f > less=%f", more, less)
	}
}
