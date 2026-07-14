package textrank

// PageRankOriginal replicates brain.js pagerank.js behavior where outgoing edge
// targets are Levenshtein distance values used as node indices.
func PageRankOriginal(graph [][]float64, linkProb, tolerance float64) []float64 {
	n := len(graph)
	if n == 0 {
		return nil
	}

	prob := make([]float64, n)
	coeff := (1 - linkProb) / float64(n)
	for i := range prob {
		prob[i] = 1.0 / float64(n)
	}

	incoming := make(map[int][]int)
	for i := 0; i < n; i++ {
		for a := 0; a < len(graph[i]); a++ {
			index := int(graph[i][a])
			incoming[index] = append(incoming[index], i)
		}
	}

	const maxIter = 1000
	for iter := 0; iter < maxIter; iter++ {
		converged := 0
		for b := 0; b < n; b++ {
			sum := 0.0
			if inc, ok := incoming[b]; ok {
				for _, a := range inc {
					ct := len(graph[a])
					if ct > 0 {
						sum += prob[a] / float64(ct)
					}
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
