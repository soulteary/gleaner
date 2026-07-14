package similarity

import "strings"

// CompareTwoStrings returns Dice coefficient similarity between two strings (0..1).
// Port of string-similarity npm package.
func CompareTwoStrings(s1, s2 string) float64 {
	if s1 == s2 {
		return 1
	}
	if len(s1) < 2 || len(s2) < 2 {
		return 0
	}

	bigrams1 := make(map[string]int)
	for i := 0; i < len(s1)-1; i++ {
		bigrams1[s1[i:i+2]]++
	}

	intersection := 0
	for i := 0; i < len(s2)-1; i++ {
		bg := s2[i : i+2]
		if count, ok := bigrams1[bg]; ok && count > 0 {
			bigrams1[bg] = count - 1
			intersection++
		}
	}
	return 2.0 * float64(intersection) / float64(len(s1)+len(s2)-2)
}

// CompareTokenStrings joins tokens with spaces then compares.
func CompareTokenStrings(tokens1, tokens2 []string) float64 {
	return CompareTwoStrings(strings.Join(tokens1, " "), strings.Join(tokens2, " "))
}
