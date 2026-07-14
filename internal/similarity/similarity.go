package similarity

import "strings"

// CompareTwoStrings returns Dice coefficient similarity between two strings (0..1).
// Port of string-similarity npm package. Bigrams are formed on rune boundaries so
// multi-byte characters (e.g. CJK) are compared as whole characters instead of
// mid-codepoint byte slices — otherwise Chinese titles yield near-random scores.
func CompareTwoStrings(s1, s2 string) float64 {
	if s1 == s2 {
		return 1
	}
	r1 := []rune(s1)
	r2 := []rune(s2)
	if len(r1) < 2 || len(r2) < 2 {
		return 0
	}

	bigrams1 := make(map[string]int)
	for i := 0; i < len(r1)-1; i++ {
		bigrams1[string(r1[i:i+2])]++
	}

	intersection := 0
	for i := 0; i < len(r2)-1; i++ {
		bg := string(r2[i : i+2])
		if count, ok := bigrams1[bg]; ok && count > 0 {
			bigrams1[bg] = count - 1
			intersection++
		}
	}
	return 2.0 * float64(intersection) / float64(len(r1)+len(r2)-2)
}

// CompareTokenStrings joins tokens with spaces then compares.
func CompareTokenStrings(tokens1, tokens2 []string) float64 {
	return CompareTwoStrings(strings.Join(tokens1, " "), strings.Join(tokens2, " "))
}
