package seed

import (
	"math/rand"
	"os"
	"testing"
)

// The generated test data has to be the same every time it is generated.
//
// It was not. Go's global rand has been auto-seeded per process since 1.20, so
// two seedings of the same code produced different children, and anything built
// on a seeded database was reproducible only by luck. The visual baselines are
// the clearest casualty: a screenshot of the attendance roster is a picture of
// this data, so the committed PNG matched or did not depending on which children
// CI happened to invent. Three consecutive CI runs showed it directly -- one
// produced a baseline, the next matched it, the third did not.

// draw takes a sample of values through the same helpers the seeder uses.
func draw(r *rand.Rand) []int {
	saved := rng
	rng = r
	defer func() { rng = saved }()

	out := make([]int, 0, 60)
	for range 20 {
		out = append(out, randInt(len(firstNames)))
		out = append(out, randInt(len(lastNames)))
		out = append(out, len(randomGender()))
	}
	return out
}

func TestGeneratedDataIsReproducible(t *testing.T) {
	first := draw(rand.New(rand.NewSource(defaultTestDataSeed)))  //nolint:gosec // determinism is the point
	second := draw(rand.New(rand.NewSource(defaultTestDataSeed))) //nolint:gosec // determinism is the point

	if len(first) != len(second) {
		t.Fatalf("sample lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("draw %d differs between runs: %d vs %d — the seeder is not reproducible",
				i, first[i], second[i])
		}
	}
}

func TestADifferentSeedGivesDifferentData(t *testing.T) {
	// The knob has to actually do something, or nobody can vary the data when
	// they want to.
	pinned := draw(rand.New(rand.NewSource(defaultTestDataSeed)))    //nolint:gosec // test data
	other := draw(rand.New(rand.NewSource(defaultTestDataSeed + 1))) //nolint:gosec // test data

	same := true
	for i := range pinned {
		if pinned[i] != other[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("a different seed produced identical data")
	}
}

func TestTestDataSeed_ReadsTheEnvironment(t *testing.T) {
	t.Setenv("SEED_RANDOM_SEED", "12345")
	if got := testDataSeed(); got != 12345 {
		t.Errorf("testDataSeed() = %d, want 12345", got)
	}
}

func TestTestDataSeed_FallsBackOnNonsense(t *testing.T) {
	// A typo in the environment must not silently reintroduce the randomness
	// this exists to remove.
	t.Setenv("SEED_RANDOM_SEED", "not-a-number")
	if got := testDataSeed(); got != defaultTestDataSeed {
		t.Errorf("testDataSeed() = %d, want the default %d", got, defaultTestDataSeed)
	}
}

func TestTestDataSeed_DefaultsWhenUnset(t *testing.T) {
	if err := os.Unsetenv("SEED_RANDOM_SEED"); err != nil {
		t.Fatalf("unset: %v", err)
	}
	if got := testDataSeed(); got != defaultTestDataSeed {
		t.Errorf("testDataSeed() = %d, want the default %d", got, defaultTestDataSeed)
	}
}
