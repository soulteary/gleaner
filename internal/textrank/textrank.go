package textrank

// Distance computes Levenshtein distance between two strings, measured in
// runes so multi-byte characters (e.g. CJK) count as a single unit.
func Distance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}
	r1 := []rune(s1)
	r2 := []rune(s2)
	if len(r1) == 0 {
		return len(r2)
	}
	if len(r2) == 0 {
		return len(r1)
	}

	prevRow := make([]int, len(r2)+1)
	for i := range prevRow {
		prevRow[i] = i
	}

	for i := 0; i < len(r1); i++ {
		nextCol := i + 1
		for j := 0; j < len(r2); j++ {
			curCol := nextCol
			cost := 1
			if r1[i] == r2[j] {
				cost = 0
			}
			nextCol = prevRow[j] + cost
			if curCol+1 < nextCol {
				nextCol = curCol + 1
			}
			if prevRow[j+1]+1 < nextCol {
				nextCol = prevRow[j+1] + 1
			}
			prevRow[j] = curCol
		}
		prevRow[len(r2)] = nextCol
	}
	return prevRow[len(r2)]
}

// SimilarityGraph builds a similarity matrix using Levenshtein distance.
func SimilarityGraph(sentences [][]string) [][]float64 {
	n := len(sentences)
	graph := make([][]float64, n)
	for i := 0; i < n; i++ {
		graph[i] = make([]float64, n)
		si := joinTokens(sentences[i])
		for j := 0; j < n; j++ {
			sj := joinTokens(sentences[j])
			graph[i][j] = float64(Distance(si, sj))
		}
	}
	return graph
}

func joinTokens(tokens []string) string {
	return stringsJoin(tokens)
}

// stringsJoin avoids importing strings in hot path for small slices - use strings.Join
func stringsJoin(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	total := len(tokens) - 1
	for _, t := range tokens {
		total += len(t)
	}
	b := make([]byte, 0, total)
	for i, t := range tokens {
		b = append(b, t...)
		if i < len(tokens)-1 {
			b = append(b, ' ')
		}
	}
	return string(b)
}

// PageRank computes weighted PageRank on a similarity graph (iterative,
// non-recursive). Each node distributes its rank to neighbours in proportion to
// the outgoing edge weight graph[a][b] (standard weighted TextRank). Treating the
// graph as weighted—rather than as unweighted boolean edges—is what lets scores
// differentiate on a dense similarity matrix; an unweighted variant degenerates
// to a uniform distribution whenever every pair of sentences is connected.
func PageRank(graph [][]float64, linkProb, tolerance float64) []float64 {
	n := len(graph)
	if n == 0 {
		return nil
	}

	prob := make([]float64, n)
	coeff := (1 - linkProb) / float64(n)
	for i := range prob {
		prob[i] = 1.0 / float64(n)
	}

	// Precompute, for each source node a, the total weight of its outgoing edges,
	// and the list of (target, weight) it points to. incoming[b] collects the
	// sources a that link into b so we can aggregate contributions per target.
	type edge struct {
		src    int
		weight float64
	}
	outWeight := make([]float64, n)
	incoming := make([][]edge, n)
	for a := 0; a < n; a++ {
		for b := 0; b < n; b++ {
			w := graph[a][b]
			if w > 0 {
				outWeight[a] += w
				incoming[b] = append(incoming[b], edge{src: a, weight: w})
			}
		}
	}

	for {
		converged := 0
		for b := 0; b < n; b++ {
			sum := 0.0
			for _, e := range incoming[b] {
				if outWeight[e.src] > 0 {
					sum += prob[e.src] * (e.weight / outWeight[e.src])
				}
			}
			res := coeff + linkProb*sum
			if res >= prob[b]-tolerance && res <= prob[b]+tolerance {
				converged++
			}
			prob[b] = res
		}
		if converged == n {
			break
		}
	}
	return prob
}
