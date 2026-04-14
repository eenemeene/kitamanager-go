// Package fuzzy provides string similarity functions for matching person names.
package fuzzy

import (
	"strings"
	"unicode/utf8"
)

// JaroWinkler computes the Jaro-Winkler similarity between two strings.
// Returns a value between 0.0 (no similarity) and 1.0 (identical).
// The algorithm was developed by Matthew Jaro and William Winkler for the
// US Census Bureau specifically for matching person names.
func JaroWinkler(s1, s2 string) float64 {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	jaro := jaroSimilarity(s1, s2)

	// Winkler modification: boost score for common prefix (up to 4 chars)
	prefixLen := 0
	for i := range min(utf8.RuneCountInString(s1), utf8.RuneCountInString(s2), 4) {
		if s1[i] != s2[i] {
			break
		}
		prefixLen = i + 1
	}

	return jaro + float64(prefixLen)*0.1*(1.0-jaro)
}

func jaroSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	len1 := utf8.RuneCountInString(s1)
	len2 := utf8.RuneCountInString(s2)
	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	// Characters from s1 and s2 are considered matching if they are the same
	// and not farther than floor(max(len1,len2)/2) - 1 apart.
	matchDist := max(len1, len2)/2 - 1
	if matchDist < 0 {
		matchDist = 0
	}

	r1 := []rune(s1)
	r2 := []rune(s2)
	s1Matches := make([]bool, len1)
	s2Matches := make([]bool, len2)

	matches := 0
	transpositions := 0

	for i := range len1 {
		lo := max(0, i-matchDist)
		hi := min(len2, i+matchDist+1)
		for j := lo; j < hi; j++ {
			if s2Matches[j] || r1[i] != r2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	k := 0
	for i := range len1 {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return (m/float64(len1) + m/float64(len2) + (m-float64(transpositions)/2)/m) / 3.0
}

// NameSimilarity computes a similarity score between a system child name and a
// bill child name. It uses token-based Jaro-Winkler matching to handle missing
// middle names, compound last names, and minor spelling variations.
//
// The score is computed by splitting both names into lowercase tokens, then for
// each token in the shorter set finding the best Jaro-Winkler match in the
// longer set. Last name matches are weighted higher than first name matches.
//
// Returns a value between 0.0 and 1.0.
func NameSimilarity(sysFirst, sysLast, billFirst, billLast string) float64 {
	sysTokens := tokenize(sysFirst + " " + sysLast)
	billTokens := tokenize(billFirst + " " + billLast)

	if len(sysTokens) == 0 || len(billTokens) == 0 {
		return 0.0
	}

	// Score last name similarity separately (weighted higher)
	lastScore := bestTokenScore(tokenize(sysLast), tokenize(billLast))

	// Score all tokens together
	shorter, longer := sysTokens, billTokens
	if len(shorter) > len(longer) {
		shorter, longer = longer, shorter
	}

	totalScore := 0.0
	used := make([]bool, len(longer))
	for _, st := range shorter {
		bestScore := 0.0
		bestIdx := -1
		for j, lt := range longer {
			if used[j] {
				continue
			}
			score := JaroWinkler(st, lt)
			if score > bestScore {
				bestScore = score
				bestIdx = j
			}
		}
		if bestIdx >= 0 {
			used[bestIdx] = true
		}
		totalScore += bestScore
	}

	// Average over shorter set — unmatched tokens in longer set don't penalize
	tokenScore := totalScore / float64(len(shorter))

	// Penalize if lengths differ significantly (many extra tokens)
	lenRatio := float64(len(shorter)) / float64(len(longer))

	// Combined: 40% last name, 40% token overlap, 20% length ratio
	return 0.4*lastScore + 0.4*tokenScore + 0.2*lenRatio
}

// bestTokenScore finds the best match score between two token sets.
func bestTokenScore(set1, set2 []string) float64 {
	if len(set1) == 0 || len(set2) == 0 {
		return 0.0
	}

	shorter, longer := set1, set2
	if len(shorter) > len(longer) {
		shorter, longer = longer, shorter
	}

	total := 0.0
	used := make([]bool, len(longer))
	for _, st := range shorter {
		best := 0.0
		bestIdx := -1
		for j, lt := range longer {
			if used[j] {
				continue
			}
			score := JaroWinkler(st, lt)
			if score > best {
				best = score
				bestIdx = j
			}
		}
		if bestIdx >= 0 {
			used[bestIdx] = true
		}
		total += best
	}
	return total / float64(len(shorter))
}

func tokenize(s string) []string {
	words := strings.Fields(strings.ToLower(s))
	result := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			result = append(result, w)
		}
	}
	return result
}
