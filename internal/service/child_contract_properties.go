package service

import (
	"cmp"
	"slices"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// validateContractProperties checks a child contract's properties against the
// funding configuration in force for it.
//
// # Why this exists
//
// Contract properties were never validated. Everything below was accepted with
// 201 against the live Berlin config, and every one of them produced a silently
// wrong monthly figure for a child worth about 1926 EUR:
//
//	{"care_type": ["ganztag","halbtag"]}  paid BOTH rates -- 1043.32 + 832.45
//	                                      - 23.00 = 1852.77, of which 832.45 is
//	                                      money nobody is owed
//	{"care_type": "ganztagg"}             matched nothing -> -23.00
//	{"care_type": "Ganztag"}              matched nothing -> -23.00
//	{}                                    matched nothing -> -23.00
//
// The -23.00 is the parent meal deduction, which applies to every contract, so
// an unmatched contract does not report zero -- it reports the deduction alone,
// which reads like a real figure.
//
// # What is checked, and what deliberately is not
//
// Three rules, all derived from the configuration rather than from this code:
//
//   - Every key/value pair must exist in the period. Catches the typo and the
//     wrong-case value.
//   - Each key carries exactly one value. The UI has always worked this way
//     ("selecting a value replaces any existing value with the same key" --
//     tag-input.tsx), so this codifies existing behaviour rather than inventing
//     a rule. Scoped to child contracts matched against funding config;
//     employee contract properties are not funding-matched and may legitimately
//     hold arrays.
//   - Every key in period.RequiredKeys is present.
//
// Age bands are deliberately NOT enforced. A care_type valid for a three-year-
// old is not valid at nine, and the child ages during the contract's life, so
// refusing at write time would reject a contract for a reason that changes by
// itself afterwards. Existence in the period is checked; age drift belongs in a
// read-time warning where it can name the months affected.
//
// # Which period
//
// The caller passes the period covering the contract's start -- or, for an
// amendment, the seam. That is the anchor the auto-apply defaults already use
// ("Anchored at the seam, not at today: a change backdated across a funding
// period boundary has to pick up the properties that applied back then"), and
// the only defensible one: validating against every period a contract spans
// would refuse it because of a future configuration change that has not
// happened yet.
//
// A nil period means no configuration covers that date. Nothing is checked,
// because there is no vocabulary to check against -- the same reason funding
// calculates to zero there.
// pathPrefix is prepended to every reported field path, so an overlay contract
// can report "add_children[0].contracts[1].properties.care_type" and name which
// of several hypothetical contracts is wrong. Empty for a real contract, whose
// form has one properties control.
func validateContractProperties(props models.ContractProperties, period *models.GovernmentFundingPeriod, pathPrefix string) error {
	if period == nil {
		return nil
	}

	// Values the configuration declares, keyed for O(1) membership tests, plus
	// the per-key value lists used to build a useful error message.
	known := make(map[string]bool, len(period.Properties))
	valuesByKey := make(map[string][]string)
	for i := range period.Properties {
		p := &period.Properties[i]
		kv := p.Key + ":" + p.Value
		if known[kv] {
			continue
		}
		known[kv] = true
		valuesByKey[p.Key] = append(valuesByKey[p.Key], p.Value)
	}

	var fields []apperror.FieldViolation

	// Walk the contract's keys in a stable order so a contract with several
	// problems reports them the same way every time.
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		values := props.GetAllValues(key)
		if len(values) > 1 {
			fields = append(fields, apperror.Field(
				"single_value", strings.Join(values, ", "),
				pathPrefix+"properties.%s", key))
			continue
		}
		if len(values) == 0 {
			// A key present with a value this accessor cannot read -- a nested
			// object, a number. Nothing can match it, so it cannot stay.
			fields = append(fields, apperror.Field(
				unknownRule(valuesByKey, key), strings.Join(valuesByKey[key], ", "),
				pathPrefix+"properties.%s", key))
			continue
		}
		if !known[key+":"+values[0]] {
			fields = append(fields, apperror.Field(
				unknownRule(valuesByKey, key), strings.Join(valuesByKey[key], ", "),
				pathPrefix+"properties.%s", key))
		}
	}

	for _, key := range period.RequiredKeys {
		if len(props.GetAllValues(key)) == 0 {
			fields = append(fields, apperror.Field(
				"one_of", strings.Join(valuesByKey[key], ", "),
				pathPrefix+"properties.%s", key))
		}
	}

	if len(fields) == 0 {
		return nil
	}
	slices.SortFunc(fields, func(a, b apperror.FieldViolation) int {
		return cmp.Compare(a.Field, b.Field)
	})
	return apperror.InvalidFields(fields...)
}

// unknownRule picks the reason that actually helps. A key the configuration
// declares, carrying a value it does not, can be answered with the permitted
// values. A key it has never heard of cannot -- listing nothing reads as
// "is not one of:" followed by empty space -- so it is named as the different
// problem it is.
func unknownRule(valuesByKey map[string][]string, key string) string {
	if len(valuesByKey[key]) == 0 {
		return "unknown_key"
	}
	return "not_one_of"
}
