package testutil

import (
	"testing"

	"github.com/soulteary/gleaner/internal/data"
	"github.com/soulteary/gleaner/internal/nnmodel"
	"github.com/soulteary/gleaner/internal/segment"
)

func openSegmenter(userDict string) (segment.Segmenter, error) {
	return segment.Default(userDict)
}

func loadAndValidateModel(modelJSON []byte) (*nnmodel.Model, error) {
	m, err := nnmodel.Load(modelJSON)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

func NewSegmenter(t *testing.T) segment.Segmenter {
	t.Helper()
	seg, err := openSegmenter(data.UserWordsTXT)
	if err != nil {
		t.Fatalf("segmenter: %v", err)
	}
	t.Cleanup(func() { seg.Close() })
	return seg
}

func NewModel(t *testing.T) *nnmodel.Model {
	t.Helper()
	m, err := loadAndValidateModel(data.SummaryJSON)
	if err != nil {
		t.Fatalf("model: %v", err)
	}
	return m
}
