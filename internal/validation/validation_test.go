package validation

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestIsWhitespaceOnly_EmptyString(t *testing.T) {
	if !IsWhitespaceOnly("") {
		t.Error("expected empty string to be whitespace only")
	}
}

func TestIsWhitespaceOnly_OnlySpaces(t *testing.T) {
	if !IsWhitespaceOnly("   ") {
		t.Error("expected spaces-only string to be whitespace only")
	}
}

func TestIsWhitespaceOnly_OnlyTabs(t *testing.T) {
	if !IsWhitespaceOnly("\t\t") {
		t.Error("expected tabs-only string to be whitespace only")
	}
}

func TestIsWhitespaceOnly_MixedWhitespace(t *testing.T) {
	if !IsWhitespaceOnly(" \t \n ") {
		t.Error("expected mixed whitespace string to be whitespace only")
	}
}

func TestIsWhitespaceOnly_ValidString(t *testing.T) {
	if IsWhitespaceOnly("test") {
		t.Error("expected valid string to not be whitespace only")
	}
}

func TestIsWhitespaceOnly_WhitespaceWithText(t *testing.T) {
	if IsWhitespaceOnly("  test  ") {
		t.Error("expected string with text to not be whitespace only")
	}
}

func TestValidateBirthdate_Past(t *testing.T) {
	pastDate := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidateBirthdate(pastDate); err != nil {
		t.Errorf("expected past date to be valid, got error: %v", err)
	}
}

func TestValidateBirthdate_Today(t *testing.T) {
	if err := ValidateBirthdate(models.Today()); err != nil {
		t.Errorf("expected today's date to be valid, got error: %v", err)
	}
}

func TestValidateBirthdate_Future(t *testing.T) {
	futureDate := models.Today().AddDate(0, 0, 1)
	if err := ValidateBirthdate(futureDate); err == nil {
		t.Error("expected future date to be invalid")
	}
}

// TestValidateBirthdate_UTCMidnightAccepted asserts the validator compares in
// UTC. On a server in a timezone east of UTC, time.Now() reports tomorrow
// locally while UTC "now" is still today — a UTC-midnight birthdate equal to
// today's UTC date must still be accepted.
func TestValidateBirthdate_UTCMidnightAccepted(t *testing.T) {
	nowUTC := time.Now().UTC()
	todayUTCMidnight := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	if err := ValidateBirthdate(todayUTCMidnight); err != nil {
		t.Errorf("expected today UTC midnight to be valid, got: %v", err)
	}
}

// Birthdate at the application timezone's calendar today is valid even when
// the input is mid-day local time (e.g., a Berlin client posting today's date
// at 14:00 Berlin = 12:00 UTC). The truncation in ValidateBirthdate must not
// promote an in-day timestamp into "tomorrow".
func TestValidateBirthdate_TodayInAppZoneNotFuture(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("Europe/Berlin must resolve: %v", err)
	}
	now := time.Now().In(berlin)
	// 14:00 in Berlin local time, today's calendar date.
	mid := time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, berlin)
	if err := ValidateBirthdate(mid); err != nil {
		t.Errorf("today @ 14:00 Berlin should be valid, got %v", err)
	}
}

func TestValidatePeriod_FromBeforeTo(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := ValidatePeriod(from, &to); err != nil {
		t.Errorf("expected from before to to be valid, got error: %v", err)
	}
}

func TestValidatePeriod_FromEqualsTo(t *testing.T) {
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidatePeriod(date, &date); err != nil {
		t.Errorf("expected same-day contract (from equals to) to be valid, got error: %v", err)
	}
}

func TestValidatePeriod_FromAfterTo(t *testing.T) {
	from := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidatePeriod(from, &to); err == nil {
		t.Error("expected from after to to be invalid")
	}
}

func TestValidatePeriod_NilTo(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidatePeriod(from, nil); err != nil {
		t.Errorf("expected nil to date to be valid, got error: %v", err)
	}
}

func TestCalculateAgeOnDate(t *testing.T) {
	tests := []struct {
		name          string
		birthdate     time.Time
		referenceDate time.Time
		expectedAge   int
	}{
		{
			name:          "exact birthday",
			birthdate:     time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			expectedAge:   5,
		},
		{
			name:          "day before birthday",
			birthdate:     time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 3, 14, 0, 0, 0, 0, time.UTC),
			expectedAge:   4,
		},
		{
			name:          "day after birthday",
			birthdate:     time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 3, 16, 0, 0, 0, 0, time.UTC),
			expectedAge:   5,
		},
		{
			name:          "newborn",
			birthdate:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			expectedAge:   0,
		},
		{
			name:          "reference date before birthdate returns 0",
			birthdate:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedAge:   0,
		},
		{
			name:          "leap year birthdate",
			birthdate:     time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
			expectedAge:   4,
		},
		{
			name:          "leap year birthdate on March 1",
			birthdate:     time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			expectedAge:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age := CalculateAgeOnDate(tt.birthdate, tt.referenceDate)
			if age != tt.expectedAge {
				t.Errorf("CalculateAgeOnDate(%v, %v) = %d, want %d",
					tt.birthdate.Format("2006-01-02"),
					tt.referenceDate.Format("2006-01-02"),
					age, tt.expectedAge)
			}
		})
	}
}

func TestFundingAgeOnDate(t *testing.T) {
	d := func(y, m, day int) time.Time {
		return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		name        string
		birthdate   time.Time
		billingDate time.Time
		expectedAge int
	}{
		// === Children born on the 1st — the only case where FundingAge differs from CalculateAge ===
		{
			name:        "born 1st, billing on birthday month — funding age is previous year",
			birthdate:   d(2023, 5, 1), // turns 2 on May 1, 2025
			billingDate: d(2025, 5, 1), // billing for May
			expectedAge: 1,             // RV-Tag: age group change on June 1, not May 1
		},
		{
			name:        "born 1st, billing month after birthday — funding age is new",
			birthdate:   d(2023, 5, 1), // turned 2 on May 1
			billingDate: d(2025, 6, 1), // billing for June
			expectedAge: 2,             // RV-Tag: age group change takes effect June 1
		},
		{
			name:        "born Jan 1, billing Jan — still previous age",
			birthdate:   d(2022, 1, 1), // turns 3 on Jan 1, 2025
			billingDate: d(2025, 1, 1),
			expectedAge: 2, // change takes effect Feb 1
		},
		{
			name:        "born Jan 1, billing Feb — new age",
			birthdate:   d(2022, 1, 1),
			billingDate: d(2025, 2, 1),
			expectedAge: 3,
		},

		// === Children born on other days — FundingAge should match CalculateAge ===
		{
			name:        "born 15th, billing before birthday month — same as CalculateAge",
			birthdate:   d(2023, 5, 15),
			billingDate: d(2025, 4, 1),
			expectedAge: 1,
		},
		{
			name:        "born 15th, billing on birthday month — same as CalculateAge",
			birthdate:   d(2023, 5, 15),
			billingDate: d(2025, 5, 1), // birthday hasn't happened yet on May 1
			expectedAge: 1,             // CalculateAge also returns 1 here
		},
		{
			name:        "born 15th, billing month after birthday — same as CalculateAge",
			birthdate:   d(2023, 5, 15),
			billingDate: d(2025, 6, 1),
			expectedAge: 2,
		},
		{
			name:        "born last day of month, billing next month — age incremented",
			birthdate:   d(2023, 4, 30),
			billingDate: d(2025, 5, 1),
			expectedAge: 2, // birthday was April 30, change takes effect May 1
		},

		// === Edge cases ===
		{
			name:        "born Feb 29 leap year, billing March 1 non-leap — age incremented",
			birthdate:   d(2020, 2, 29),
			billingDate: d(2025, 3, 1), // Feb 28 is the ref date; birthday Feb 29 hasn't occurred
			expectedAge: 4,             // Feb 28, 2025 → birthday hasn't happened in 2025 yet
		},
		{
			name:        "born Feb 29, billing March 1 leap year — age incremented",
			birthdate:   d(2020, 2, 29),
			billingDate: d(2024, 3, 1), // Feb 29 is the ref date in leap year; birthday happens
			expectedAge: 4,
		},
		{
			name:        "billing date before birth — age 0",
			birthdate:   d(2025, 6, 1),
			billingDate: d(2025, 1, 1),
			expectedAge: 0,
		},
		{
			name:        "same month and year as birth, born on 1st — age 0",
			birthdate:   d(2025, 5, 1),
			billingDate: d(2025, 5, 1),
			expectedAge: 0, // born today, change happens June 1
		},
		{
			// Max Kaphamel: born Sept 1, 2022. Billing Sept 2024.
			// CalculateAge would say 2 (birthday today). FundingAge should say 1.
			name:        "Max Kaphamel real case — born Sept 1, billing Sept",
			birthdate:   d(2022, 9, 1),
			billingDate: d(2024, 9, 1),
			expectedAge: 1, // age group change on Oct 1
		},
		{
			name:        "Max Kaphamel — billing Oct, age 2",
			birthdate:   d(2022, 9, 1),
			billingDate: d(2024, 10, 1),
			expectedAge: 2, // age group change took effect Oct 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age := FundingAgeOnDate(tt.birthdate, tt.billingDate)
			if age != tt.expectedAge {
				t.Errorf("FundingAgeOnDate(%s, %s) = %d, want %d",
					tt.birthdate.Format("2006-01-02"),
					tt.billingDate.Format("2006-01-02"),
					age, tt.expectedAge)
			}
		})
	}
}

// TestFundingAgeOnDate_WarnsOnNonFirstOfMonth asserts that callers who
// accidentally pass a non-first-of-month date get a log signal. The RV-Tag
// semantics only apply to first-of-month billing; mid-month calls compute
// "age as of yesterday" which is surprising.
func TestFundingAgeOnDate_WarnsOnNonFirstOfMonth(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(oldLogger)

	birthdate := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC)
	_ = FundingAgeOnDate(birthdate, mid)

	if !bytes.Contains(buf.Bytes(), []byte("non-first-of-month")) {
		t.Errorf("expected warn log about non-first-of-month, got: %s", buf.String())
	}
}

// TestFundingAgeOnDate_NoWarnOnFirstOfMonth asserts the normal path produces
// no log noise for first-of-month billing.
func TestFundingAgeOnDate_NoWarnOnFirstOfMonth(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(oldLogger)

	birthdate := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)
	_ = FundingAgeOnDate(birthdate, time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC))
	if bytes.Contains(buf.Bytes(), []byte("non-first-of-month")) {
		t.Errorf("no warn expected for first-of-month billing, got: %s", buf.String())
	}
}

func TestFundingAgeOnDate_MatchesCalculateAge_ForNonFirstBirthdays(t *testing.T) {
	// For children NOT born on the 1st, FundingAgeOnDate should always match
	// CalculateAgeOnDate when the billing date is the 1st of the month
	d := func(y, m, day int) time.Time {
		return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
	}

	for day := 2; day <= 28; day++ {
		birthdate := d(2020, 6, day)
		for month := 1; month <= 12; month++ {
			billingDate := d(2025, month, 1)
			funding := FundingAgeOnDate(birthdate, billingDate)
			actual := CalculateAgeOnDate(birthdate, billingDate)
			if funding != actual {
				t.Errorf("day=%d, billing=%s: FundingAge=%d != CalculateAge=%d",
					day, billingDate.Format("2006-01-02"), funding, actual)
			}
		}
	}
}
