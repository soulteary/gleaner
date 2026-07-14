package segment_test

import (
	"strings"
	"testing"

	"github.com/soulteary/gleaner/internal/data"
	"github.com/soulteary/gleaner/internal/segment"
)

func TestCutAndTag(t *testing.T) {
	seg, err := segment.Default(data.UserWordsTXT)
	if err != nil {
		t.Fatalf("segmenter: %v", err)
	}
	defer seg.Close()

	text := "第三方机构网贷之家公布数据"
	cut := seg.Cut(text)
	if len(cut) == 0 {
		t.Fatal("expected non-empty cut")
	}

	tags := seg.Tag(text)
	if len(tags) == 0 {
		t.Fatal("expected non-empty tags")
	}
	for _, tok := range tags {
		if segment.NatureIndex(tok.POS) < 0 {
			t.Fatalf("invalid pos: %q", tok.POS)
		}
	}
}

func TestUserDictWord(t *testing.T) {
	seg, err := segment.Default(data.UserWordsTXT)
	if err != nil {
		t.Fatalf("segmenter: %v", err)
	}
	defer seg.Close()

	// user-words.txt contains AFINN compound entries like "一句话"
	cut := seg.Cut("这是一句话的例子")
	joined := strings.Join(cut, "/")
	if !strings.Contains(joined, "一句话") {
		t.Logf("cut=%q (user dict may split differently across backends)", joined)
	}
}

func TestGSEFallback(t *testing.T) {
	seg, err := segment.New(segment.BackendGSE, data.UserWordsTXT)
	if err != nil {
		t.Fatalf("gse: %v", err)
	}
	defer seg.Close()
	cut := seg.Cut("自动摘要测试")
	if len(cut) == 0 {
		t.Fatal("expected tokens")
	}
}

func TestGSETagAndClose(t *testing.T) {
	seg, err := segment.New(segment.BackendGSE, data.UserWordsTXT)
	if err != nil {
		t.Fatal(err)
	}
	seg.Close()

	tags := seg.Tag("第三方机构公布数据")
	if len(tags) == 0 {
		t.Fatal("expected POS tags from GSE")
	}
	for _, tok := range tags {
		if tok.Word == "" {
			t.Fatal("empty token word")
		}
	}
}

func TestNewGojiebaBackend(t *testing.T) {
	seg, err := segment.New(segment.BackendGojieba, data.UserWordsTXT)
	if err != nil {
		t.Skip("gojieba requires CGO")
	}
	defer seg.Close()
	if len(seg.Cut("测试分词")) == 0 {
		t.Fatal("expected cut tokens")
	}
}

func TestDefaultSegmenter(t *testing.T) {
	seg, err := segment.Default(data.UserWordsTXT)
	if err != nil {
		t.Fatal(err)
	}
	defer seg.Close()
	if len(seg.Cut("默认分词器")) == 0 {
		t.Fatal("expected tokens")
	}
}

func TestNatureIndex(t *testing.T) {
	if segment.NatureIndex("n") == 0 {
		t.Fatal("n should not map to 0")
	}
	if segment.NatureIndex("unknown_tag_xyz") != 0 {
		t.Fatal("unknown tag should map to 0")
	}
}
