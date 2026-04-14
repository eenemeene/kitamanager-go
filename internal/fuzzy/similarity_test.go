package fuzzy

import (
	"math"
	"testing"
)

// approx checks that two floats are within epsilon of each other.
func approx(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

// ============================================================
// JaroWinkler unit tests
// ============================================================

func TestJaroWinkler_IdenticalStrings(t *testing.T) {
	if score := JaroWinkler("hello", "hello"); score != 1.0 {
		t.Errorf("identical strings: got %f, want 1.0", score)
	}
}

func TestJaroWinkler_EmptyStrings(t *testing.T) {
	if score := JaroWinkler("", ""); score != 1.0 {
		t.Errorf("both empty: got %f, want 1.0", score)
	}
	if score := JaroWinkler("", "abc"); score != 0.0 {
		t.Errorf("one empty: got %f, want 0.0", score)
	}
	if score := JaroWinkler("abc", ""); score != 0.0 {
		t.Errorf("other empty: got %f, want 0.0", score)
	}
}

func TestJaroWinkler_SingleChars(t *testing.T) {
	if score := JaroWinkler("a", "a"); score != 1.0 {
		t.Errorf("same single char: got %f, want 1.0", score)
	}
	if score := JaroWinkler("a", "b"); score != 0.0 {
		t.Errorf("different single chars: got %f, want 0.0", score)
	}
}

func TestJaroWinkler_ClassicExample(t *testing.T) {
	// "martha" vs "marhta" is a well-known Jaro-Winkler reference case
	score := JaroWinkler("martha", "marhta")
	if !approx(score, 0.961, 0.01) {
		t.Errorf("martha/marhta: got %f, want ~0.961", score)
	}
}

func TestJaroWinkler_CaseInsensitive(t *testing.T) {
	if score := JaroWinkler("Hello", "hello"); score != 1.0 {
		t.Errorf("case insensitive: got %f, want 1.0", score)
	}
}

func TestJaroWinkler_SimilarNames(t *testing.T) {
	tests := []struct {
		s1, s2 string
		minExp float64
	}{
		{"johannes", "johan", 0.85},     // prefix match
		{"katarina", "katharina", 0.95}, // one char insertion
		{"lukas", "lucas", 0.88},        // single char substitution
		{"schmidt", "schmitt", 0.93},    // common German last name variant
		{"mueller", "müller", 0.70},     // umlaut (different bytes)
	}
	for _, tt := range tests {
		score := JaroWinkler(tt.s1, tt.s2)
		if score < tt.minExp {
			t.Errorf("JaroWinkler(%q, %q) = %f, want >= %f", tt.s1, tt.s2, score, tt.minExp)
		}
	}
}

func TestJaroWinkler_DissimilarNames(t *testing.T) {
	tests := []struct {
		s1, s2 string
		maxExp float64
	}{
		{"anna", "tobias", 0.50},
		{"berger", "klein", 0.55},
		{"xyz", "abc", 0.50},
	}
	for _, tt := range tests {
		score := JaroWinkler(tt.s1, tt.s2)
		if score > tt.maxExp {
			t.Errorf("JaroWinkler(%q, %q) = %f, want <= %f", tt.s1, tt.s2, score, tt.maxExp)
		}
	}
}

// ============================================================
// NameSimilarity unit tests
// ============================================================

func TestNameSimilarity_ExactMatch(t *testing.T) {
	score := NameSimilarity("Felix", "Berger", "Felix", "Berger")
	if !approx(score, 1.0, 0.01) {
		t.Errorf("exact match: got %f, want ~1.0", score)
	}
}

func TestNameSimilarity_MissingMiddleName(t *testing.T) {
	// System: "Anna Berger", Bill: "Anna Lena Berger"
	// The extra middle name should still score high
	score := NameSimilarity("Anna", "Berger", "Anna Lena", "Berger")
	if score < 0.75 {
		t.Errorf("missing middle name: got %f, want >= 0.75", score)
	}
}

func TestNameSimilarity_MissingMiddleName_Reverse(t *testing.T) {
	// System has middle name, bill doesn't
	score := NameSimilarity("Anna Lena", "Berger", "Anna", "Berger")
	if score < 0.75 {
		t.Errorf("extra middle name in system: got %f, want >= 0.75", score)
	}
}

func TestNameSimilarity_DifferentFirstNameSimilarSpelling(t *testing.T) {
	// System: "Katarina Fischer", Bill: "Katharina Fischer"
	// One char difference in first name
	score := NameSimilarity("Katarina", "Fischer", "Katharina", "Fischer")
	if score < 0.85 {
		t.Errorf("similar first name spelling: got %f, want >= 0.85", score)
	}
}

func TestNameSimilarity_CompoundLastName(t *testing.T) {
	// System: "Jonas Weber", Bill: "Jonas Richter Weber"
	// Extra token in last name — should still match well
	score := NameSimilarity("Jonas", "Weber", "Jonas", "Richter Weber")
	if score < 0.70 {
		t.Errorf("compound last name: got %f, want >= 0.70", score)
	}
}

func TestNameSimilarity_CompoundLastNameWithPrefix(t *testing.T) {
	// System: "Lena Graf", Bill: "Lena von der Graf"
	// German nobility prefix in last name
	score := NameSimilarity("Lena", "Graf", "Lena", "von der Graf")
	if score < 0.65 {
		t.Errorf("last name with prefix: got %f, want >= 0.65", score)
	}
}

func TestNameSimilarity_ExtraFirstAndLastTokens(t *testing.T) {
	// System: "Tom Klein", Bill: "Tom Rio Conde Klein"
	// Extra middle name AND compound last name
	score := NameSimilarity("Tom", "Klein", "Tom Rio", "Conde Klein")
	if score < 0.60 {
		t.Errorf("extra tokens in both names: got %f, want >= 0.60", score)
	}
}

func TestNameSimilarity_CompletelyDifferent(t *testing.T) {
	// Totally different names — should score low
	score := NameSimilarity("Felix", "Berger", "Maria", "Schmidt")
	if score > 0.50 {
		t.Errorf("completely different: got %f, want < 0.50", score)
	}
}

func TestNameSimilarity_SameLastNameDifferentFirst(t *testing.T) {
	// Same last name but different first name — scores moderately high because
	// last name is weighted 40%. The birth month/year hard filter in the matching
	// logic will prevent false matches between siblings.
	score := NameSimilarity("Felix", "Berger", "Tobias", "Berger")
	if score < 0.45 {
		t.Errorf("same last different first: got %f, want >= 0.45 (last name match)", score)
	}
	// But should score lower than an actual match with same first name
	exactScore := NameSimilarity("Felix", "Berger", "Felix", "Berger")
	if score >= exactScore {
		t.Errorf("different first should score lower than exact: same=%f, different=%f", exactScore, score)
	}
}

func TestNameSimilarity_SameFirstNameDifferentLast(t *testing.T) {
	// Same first name, different last name
	score := NameSimilarity("Felix", "Berger", "Felix", "Schmidt")
	// First name match helps but last name mismatch should lower the score
	if score > 0.75 {
		t.Errorf("same first different last: got %f, want < 0.75", score)
	}
}

func TestNameSimilarity_EmptyNames(t *testing.T) {
	score := NameSimilarity("", "", "", "")
	if score != 0.0 {
		t.Errorf("all empty: got %f, want 0.0", score)
	}
	score = NameSimilarity("Felix", "Berger", "", "")
	if score != 0.0 {
		t.Errorf("bill empty: got %f, want 0.0", score)
	}
}

func TestNameSimilarity_TwoCloseMatches(t *testing.T) {
	// Two children in system with similar names — which one scores higher?
	// System child 1: "Anna Berger"
	// System child 2: "Anna Lena Berger"
	// Bill: "Berger, Anna Lena"
	score1 := NameSimilarity("Anna", "Berger", "Anna Lena", "Berger")
	score2 := NameSimilarity("Anna Lena", "Berger", "Anna Lena", "Berger")
	if score2 <= score1 {
		t.Errorf("exact match should score higher: score1=%f (Anna Berger), score2=%f (Anna Lena Berger)", score1, score2)
	}
	if score2 < 0.95 {
		t.Errorf("exact match should be very high: got %f", score2)
	}
}

func TestNameSimilarity_TwoCloseMatches_DifferentLastNames(t *testing.T) {
	// Two system children could match — one with right last name scores higher
	// System child 1: "Jonas Weber"
	// System child 2: "Jonas Richter"
	// Bill: "Weber, Jonas"
	score1 := NameSimilarity("Jonas", "Weber", "Jonas", "Weber")
	score2 := NameSimilarity("Jonas", "Richter", "Jonas", "Weber")
	if score2 >= score1 {
		t.Errorf("correct last name should score higher: score1=%f (Weber), score2=%f (Richter)", score1, score2)
	}
}

func TestNameSimilarity_Symmetric(t *testing.T) {
	// Swapping system and bill should give the same score
	score1 := NameSimilarity("Anna Lena", "Berger", "Anna", "Berger")
	score2 := NameSimilarity("Anna", "Berger", "Anna Lena", "Berger")
	if !approx(score1, score2, 0.01) {
		t.Errorf("asymmetric: NameSimilarity(sys→bill)=%f, NameSimilarity(bill→sys)=%f", score1, score2)
	}
}

func TestNameSimilarity_TypoInFirstName(t *testing.T) {
	// "Johaness" vs "Johannes" — common typo
	score := NameSimilarity("Johaness", "Weber", "Johannes", "Weber")
	if score < 0.85 {
		t.Errorf("typo in first name: got %f, want >= 0.85", score)
	}
}
