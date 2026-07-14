package sentiment

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	"github.com/soulteary/gleaner/internal/segment"
)

var particleRe = regexp.MustCompile(`[的得着了过]$`)

// Result holds sentiment analysis output.
type Result struct {
	Score       int      `json:"score"`
	Comparative float64  `json:"comparative"`
	Tokens      []string `json:"tokens"`
	Words       []string `json:"words"`
	Positive    []string `json:"positive"`
	Negative    []string `json:"negative"`
}

// Analyzer performs AFINN-based Chinese sentiment analysis.
type Analyzer struct {
	seg        segment.Segmenter
	afinn      map[string]int
	protection []string
	mu         sync.RWMutex
}

// New creates a sentiment analyzer.
func New(seg segment.Segmenter, afinnJSON, protectionJSON []byte) (*Analyzer, error) {
	var afinn map[string]int
	if err := json.Unmarshal(afinnJSON, &afinn); err != nil {
		return nil, err
	}
	var protection []string
	if err := json.Unmarshal(protectionJSON, &protection); err != nil {
		return nil, err
	}
	return &Analyzer{
		seg:        seg,
		afinn:      afinn,
		protection: protection,
	}, nil
}

// Analyze returns sentiment for the given phrase.
func (a *Analyzer) Analyze(phrase string, inject map[string]int) Result {
	a.mu.RLock()
	lexicon := a.afinn
	a.mu.RUnlock()

	if inject != nil {
		merged := make(map[string]int, len(lexicon)+len(inject))
		for k, v := range lexicon {
			merged[k] = v
		}
		for k, v := range inject {
			merged[k] = v
		}
		lexicon = merged
	}

	tokens := a.tokenize(phrase)
	score := 0
	var words, positive, negative []string

	for _, tok := range tokens {
		item, ok := lexicon[tok]
		if !ok {
			continue
		}
		words = append(words, tok)
		if item > 0 {
			positive = append(positive, tok)
		}
		if item < 0 {
			negative = append(negative, tok)
		}
		score += item
	}

	comparative := 0.0
	if len(tokens) > 0 {
		comparative = float64(score) / float64(len(tokens))
	}

	return Result{
		Score:       score,
		Comparative: comparative,
		Tokens:      tokens,
		Words:       words,
		Positive:    positive,
		Negative:    negative,
	}
}

func (a *Analyzer) tokenize(input string) []string {
	segTokens := a.seg.Cut(input)
	var out []string
	for i, tok := range segTokens {
		it := particleRe.ReplaceAllString(tok, "")
		if i >= 1 {
			pair := segTokens[i-1] + tok
			if contains(a.protection, pair) {
				if len(out) > 0 {
					out = out[:len(out)-1]
				}
				out = append(out, pair)
				continue
			}
		}
		if contains(a.protection, tok) {
			out = append(out, tok)
		} else if it != "" {
			out = append(out, it)
		}
	}
	return out
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// Score returns only the numeric sentiment score.
func (a *Analyzer) Score(phrase string) int {
	return a.Analyze(phrase, nil).Score
}

// MergeInject merges custom lexicon entries (mutates analyzer copy semantics via inject param).
func MergeInject(base map[string]int, inject map[string]int) map[string]int {
	if inject == nil {
		return base
	}
	merged := make(map[string]int, len(base)+len(inject))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range inject {
		merged[strings.TrimSpace(k)] = v
	}
	return merged
}
