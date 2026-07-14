package testutil

import (
	"testing"

	"github.com/soulteary/gleaner/internal/data"
	"github.com/soulteary/gleaner/internal/nnmodel"
	"github.com/soulteary/gleaner/internal/segment"
)

func TestNewSegmenter(t *testing.T) {
	seg := NewSegmenter(t)
	if len(seg.Cut("测试")) == 0 {
		t.Fatal("expected tokens")
	}
}

func TestNewModel(t *testing.T) {
	m := NewModel(t)
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenSegmenter(t *testing.T) {
	seg, err := openSegmenter(data.UserWordsTXT)
	if err != nil {
		t.Fatal(err)
	}
	defer seg.Close()
}

func TestLoadAndValidateModel(t *testing.T) {
	m, err := loadAndValidateModel(data.SummaryJSON)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAndValidateModelErrors(t *testing.T) {
	if _, err := loadAndValidateModel([]byte(`{invalid`)); err == nil {
		t.Fatal("expected load error")
	}
	if _, err := loadAndValidateModel([]byte(`{"sizes":[1],"layers":[]}`)); err == nil {
		t.Fatal("expected validate error")
	}
}

func TestSegmentNewDirect(t *testing.T) {
	seg, err := segment.New(segment.BackendGSE, data.UserWordsTXT)
	if err != nil {
		t.Fatal(err)
	}
	defer seg.Close()
}

func TestNewModelViaNNModelLoad(t *testing.T) {
	m, err := nnmodel.Load(data.SummaryJSON)
	if err != nil {
		t.Fatal(err)
	}
	_ = m
}
