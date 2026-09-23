package seed

import (
	"slices"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// The employee table used to carry fixed contract start dates, which quietly
// rotted: years of service is measured against the real clock, so every
// employee crept up the TVöD ladder as time passed and nothing ever bumped
// their step to clear the promotion. Seeded in 2024 the step-promotion page
// showed two people due; by late 2026 it showed the same two plus whoever had
// since drifted past a threshold, and the cohort comments ("Upcoming
// employees") had started to lie.
//
// The dates are offsets from today now. These tests are what makes that hold:
// they run the seeded staff through the same eligibility rule the service
// uses, at a year's worth of start dates plus a couple of far-future ones, and
// insist the answer never moves.

// dueForPromotion returns the employees the step-promotion page would list on
// the given date, by the same rule StepPromotionService applies: eligibility
// from years of service since the earliest contract, against the ladder the
// seeder actually writes.
func dueForPromotion(t *testing.T, now time.Time) []string {
	t.Helper()

	entries := tvoedSuEEntries()
	var due []string

	for _, e := range employeeDefs(now) {
		contracts := make([]models.EmployeeContract, 0, len(e.contracts))
		for _, c := range e.contracts {
			contracts = append(contracts, models.EmployeeContract{
				BaseContract: models.BaseContract{Period: models.Period{From: c.from, To: c.to}},
				Grade:        c.grade,
				Step:         c.step,
			})
		}

		active := -1
		for i := range contracts {
			if contracts[i].IsActiveOn(now) {
				active = i
				break
			}
		}
		if active < 0 {
			continue
		}

		years := service.CalculateYearsOfService(contracts, now)
		eligible := service.DetermineEligibleStep(years, entries, contracts[active].Grade)
		if eligible > contracts[active].Step {
			due = append(due, e.firstName+" "+e.lastName)
		}
	}

	slices.Sort(due)
	return due
}

// wantDue is the picture the table is written to produce: three promotions,
// two grades, one of them part-time so the pro-rata arithmetic is exercised.
var wantDue = []string{"Inge Schwarz", "Martin Becker", "Thomas Schmidt"}

func TestStepPromotionsAreStableOverTime(t *testing.T) {
	// A year of consecutive days covers every month-length and leap-day case
	// AddDate can normalize into; the far dates confirm nothing is anchored to
	// a particular decade.
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for day := range 366 {
		now := start.AddDate(0, 0, day)
		if got := dueForPromotion(t, now); !slices.Equal(got, wantDue) {
			t.Errorf("on %s: due for promotion = %v, want %v", now.Format(models.DateFormat), got, wantDue)
		}
	}

	for _, now := range []time.Time{
		time.Date(2029, time.February, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2032, time.February, 29, 0, 0, 0, 0, time.UTC),
		time.Date(2040, time.August, 31, 0, 0, 0, 0, time.UTC),
	} {
		if got := dueForPromotion(t, now); !slices.Equal(got, wantDue) {
			t.Errorf("on %s: due for promotion = %v, want %v", now.Format(models.DateFormat), got, wantDue)
		}
	}
}

// The cohort comments in the table are load-bearing documentation, so they get
// asserted rather than trusted: active staff active, leavers gone, starters
// still to come, whenever the seeder runs.
func TestEmployeeCohortsHoldTheirMeaning(t *testing.T) {
	former := map[string]bool{"Jürgen Lang": true, "Wolfgang Krüger": true, "Renate Meier": true}
	upcoming := map[string]bool{"Lena Hofmann": true, "Sophie Lehmann": true}

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for day := range 366 {
		now := start.AddDate(0, 0, day)

		for _, e := range employeeDefs(now) {
			name := e.firstName + " " + e.lastName

			active, starts := false, false
			for _, c := range e.contracts {
				contract := models.EmployeeContract{
					BaseContract: models.BaseContract{Period: models.Period{From: c.from, To: c.to}},
				}
				if contract.IsActiveOn(now) {
					active = true
				}
				if c.from.After(now) {
					starts = true
				}
			}

			switch {
			case former[name]:
				if active || starts {
					t.Errorf("on %s: %s is seeded as a former employee but is active=%v, upcoming=%v",
						now.Format(models.DateFormat), name, active, starts)
				}
			case upcoming[name]:
				if active || !starts {
					t.Errorf("on %s: %s is seeded as an upcoming employee but is active=%v, upcoming=%v",
						now.Format(models.DateFormat), name, active, starts)
				}
			default:
				if !active {
					t.Errorf("on %s: %s is seeded as active staff but has no active contract",
						now.Format(models.DateFormat), name)
				}
			}
		}
	}
}

// Every graded contract has to name a grade the pay plan actually pays, or the
// employee silently costs nothing in the salary and forecast views.
func TestSeededGradesExistInThePayPlan(t *testing.T) {
	entries := tvoedSuEEntries()

	for _, e := range employeeDefs(time.Now()) {
		for _, c := range e.contracts {
			if c.grade == "Minijob" { // its own single-entry pay plan
				continue
			}
			found := slices.ContainsFunc(entries, func(entry models.PayPlanEntry) bool {
				return entry.Grade == c.grade && entry.Step == c.step
			})
			if !found {
				t.Errorf("%s %s: contract is %s step %d, which the pay plan does not price",
					e.firstName, e.lastName, c.grade, c.step)
			}
		}
	}
}
