package nnmodel

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/soulteary/gleaner/internal/segment"
)

const FeatureSize = 500

// Model is a feed-forward neural network loaded from brain.js JSON.
type Model struct {
	Sizes  []int
	Layers []map[string]nodeWeights
}

type nodeWeights struct {
	Bias    float64            `json:"bias"`
	Weights map[string]float64 `json:"weights"`
}

type brainJSON struct {
	Sizes      []int                    `json:"sizes"`
	Layers     []map[string]nodeWeights `json:"layers"`
	Activation string                   `json:"activation"`
}

var (
	defaultModel *Model
	loadOnce     sync.Once
	loadErr      error
)

// Load parses brain.js model JSON.
func Load(data []byte) (*Model, error) {
	var raw brainJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &Model{Sizes: raw.Sizes, Layers: raw.Layers}, nil
}

// DefaultModel returns the singleton model loaded from embedded data.
func DefaultModel(modelJSON []byte) (*Model, error) {
	loadOnce.Do(func() {
		defaultModel, loadErr = Load(modelJSON)
	})
	return defaultModel, loadErr
}

// GenerateFeature builds a 500-dim POS feature vector for a sentence.
func GenerateFeature(seg segment.Segmenter, text string) []float64 {
	arr := make([]float64, FeatureSize)
	tokens := seg.Tag(text)
	for i, tok := range tokens {
		if i >= FeatureSize {
			break
		}
		arr[i] = float64(segment.NatureIndex(tok.POS))
	}
	return arr
}

// Rank returns the model score for a sentence (0..1).
func (m *Model) Rank(seg segment.Segmenter, text string) float64 {
	feature := GenerateFeature(seg, text)
	out := m.forward(feature)
	if len(out) == 0 {
		return 0
	}
	return out[0]
}

func (m *Model) forward(input []float64) []float64 {
	current := input
	for layerIdx := 1; layerIdx < len(m.Layers); layerIdx++ {
		curLayer := m.Layers[layerIdx]
		next := make([]float64, len(curLayer))
		for nodeIdx := 0; nodeIdx < len(curLayer); nodeIdx++ {
			nodeKey := strconv.Itoa(nodeIdx)
			node := curLayer[nodeKey]
			sum := node.Bias
			for wKey, w := range node.Weights {
				wi, _ := strconv.Atoi(wKey)
				if wi < len(current) {
					sum += w * current[wi]
				}
			}
			next[nodeIdx] = sigmoid(sum)
		}
		current = next
	}
	return current
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// Validate checks model structure matches expected architecture.
func (m *Model) Validate() error {
	expected := []int{500, 200, 100, 50, 25, 5, 1}
	if len(m.Sizes) != len(expected) {
		return fmt.Errorf("unexpected sizes: %v", m.Sizes)
	}
	for i, v := range expected {
		if m.Sizes[i] != v {
			return fmt.Errorf("size[%d]=%d, want %d", i, m.Sizes[i], v)
		}
	}
	return nil
}
