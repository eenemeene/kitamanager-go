/**
 * School-enrollment (Einschulung) rules per German Bundesland.
 *
 * Berlin rule (https://familienportal.berlin.de/artikel/einschulung-frueher-oder-spaeter):
 *   - Muss-Kind: turns 6 on or before Sep 30 → school starts that August.
 *   - Kann-Kind: turns 6 between Oct 1 and Mar 31 of the following calendar
 *     year → parents may apply for early enrollment the preceding August.
 */
import { getDayBefore, toUTCDate, todayBerlin } from './contracts';

type StateRules = {
  stichtag: { month: number; day: number };
  kannWindowEnd: { month: number; day: number };
};

const rulesByState: Record<string, StateRules> = {
  berlin: {
    stichtag: { month: 9, day: 30 },
    kannWindowEnd: { month: 3, day: 31 },
  },
};

// Fallback used only by calculateContractEndDate for backward compatibility
// with organizations whose state isn't yet modelled. Other callers should use
// classifySchoolEnrollment and treat an unknown state as "no rule".
const defaultStichtag = { month: 9, day: 30 };

/**
 * Suggested Kita contract end date for a child: July 31 of the Muss-school-year.
 * Leniently falls back to a Sep 30 Stichtag for unknown states.
 *
 * @returns YYYY-MM-DD or null if inputs are unusable
 */
export function calculateContractEndDate(birthdate: string, state: string): string | null {
  if (!birthdate || !state) return null;
  const bd = parseBirthdate(birthdate);
  if (!bd) return null;
  const stichtag = rulesByState[state]?.stichtag ?? defaultStichtag;
  return `${computeMussYear(bd, stichtag)}-07-31`;
}

/**
 * The contract end date to *offer* in a create form, or '' for no suggestion.
 *
 * calculateContractEndDate answers "when does this child's Kita time end on
 * paper", which is a fixed consequence of their birthdate and stays in the past
 * once it has passed. Offering it as the end of a contract that has not started
 * yet produces `to` < `from` -- a form that opens already invalid, with the
 * schema refusing a value the form itself filled in.
 *
 * Reachable well beyond the deferral case: a section may carry no upper age
 * limit at all (the seeded "Grosse" has max_age_months = null), so a Kita can
 * legitimately hold children past their school start, and every new contract
 * for one of them hits this.
 *
 * `notBefore` is the start date the suggestion will sit next to when the caller
 * knows it; callers that leave `from` empty for the user to fill get today,
 * which is the earliest start they are realistically about to type. A date that
 * is already past is not a suggestion, so return nothing and let them choose --
 * an open-ended contract past the school start now raises its own warning in
 * the children table.
 */
export function suggestContractEnd(
  birthdate: string,
  state: string,
  notBefore?: string,
  schoolEntryDate?: string | null
): string {
  // A recorded entry date wins: the child leaves the day before it, whatever
  // the birthdate would have implied.
  const suggested = schoolEntryDate
    ? getDayBefore(schoolEntryDate.slice(0, 10))
    : calculateContractEndDate(birthdate, state);
  if (!suggested) return '';
  const floor = notBefore ? toUTCDate(notBefore) : todayBerlin();
  return toUTCDate(suggested) < floor ? '' : suggested;
}

export type SchoolEnrollment = {
  /**
   * True when these dates come from a recorded school-entry date rather than
   * from the birthdate -- a Zurückstellung, most often. Worth surfacing rather
   * than silently showing the new year: it is the difference between "this is
   * when the rule says" and "somebody was told otherwise".
   */
  overridden: boolean;
  /**
   * The year the birthdate alone implies, kept alongside an override so a caller
   * can say which way it diverges -- later is a Zurückstellung, earlier an early
   * entry. Null when the state has no rules to derive it from, which is the one
   * case where an override tells us the date without telling us the reason.
   */
  computedMussYear: number | null;
  /** Calendar year of the August school-start where the child is a Muss-Kind. */
  mussYear: number;
  /** Calendar year of the earlier August school-start where the child may apply as Kann-Kind, or null. */
  kannYear: number | null;
  /** YYYY-07-31 — Kita contract end date for the Muss-school-year. */
  mussContractEnd: string;
  /** YYYY-07-31 for the Kann-school-year, or null if not Kann-eligible. */
  kannContractEnd: string | null;
};

/**
 * Classify a child's school enrollment status for the given organization state.
 *
 * Strict on unknown states (returns null) — the caller should hide any
 * Muss/Kann UI rather than display a guessed category.
 */
export function classifySchoolEnrollment(
  birthdate: string,
  state: string,
  schoolEntryDate?: string | null
): SchoolEnrollment | null {
  // A recorded entry date answers the question outright, so it is checked
  // before the state rules and works even for a state we have no rules for --
  // knowing the date does not depend on being able to derive it.
  const override = schoolEntryDate ? parseBirthdate(schoolEntryDate) : null;
  if (override) {
    const bd = parseBirthdate(birthdate);
    const rules = rulesByState[state];
    return {
      overridden: true,
      computedMussYear: bd && rules ? computeMussYear(bd, rules.stichtag) : null,
      mussYear: override.getFullYear(),
      // Kann is a question about which year the child *could* start. Once a
      // decision is on file it is answered, so offering the alternative is noise.
      kannYear: null,
      mussContractEnd: getDayBefore(schoolEntryDate!.slice(0, 10)),
      kannContractEnd: null,
    };
  }

  if (!birthdate || !state) return null;
  const bd = parseBirthdate(birthdate);
  if (!bd) return null;
  const rules = rulesByState[state];
  if (!rules) return null;

  const mussYear = computeMussYear(bd, rules.stichtag);
  const kannYear = computeKannYear(bd, rules, mussYear);

  return {
    overridden: false,
    computedMussYear: mussYear,
    mussYear,
    kannYear,
    mussContractEnd: `${mussYear}-07-31`,
    kannContractEnd: kannYear !== null ? `${kannYear}-07-31` : null,
  };
}

function parseBirthdate(s: string): Date | null {
  // Accept both "YYYY-MM-DD" (form input) and "YYYY-MM-DDTHH:MM:SSZ" (API response).
  // Slicing to 10 chars pins the date in the local timezone and avoids the UTC-midnight
  // off-by-one that `new Date("...Z").getDate()` would cause east of UTC.
  if (s.length < 10) return null;
  const d = new Date(`${s.slice(0, 10)}T00:00:00`);
  return isNaN(d.getTime()) ? null : d;
}

function computeMussYear(bd: Date, stichtag: { month: number; day: number }): number {
  const turnsSixYear = bd.getFullYear() + 6;
  const sixthBirthday = new Date(turnsSixYear, bd.getMonth(), bd.getDate());
  const stichtagDate = new Date(turnsSixYear, stichtag.month - 1, stichtag.day);
  return sixthBirthday <= stichtagDate ? turnsSixYear : turnsSixYear + 1;
}

function computeKannYear(bd: Date, rules: StateRules, mussYear: number): number | null {
  const kannYear = mussYear - 1;
  const sixthBirthday = new Date(bd.getFullYear() + 6, bd.getMonth(), bd.getDate());
  // Open on the left (Stichtag itself would be a Muss-Kind), inclusive on the right.
  const windowStart = new Date(kannYear, rules.stichtag.month - 1, rules.stichtag.day);
  const windowEnd = new Date(kannYear + 1, rules.kannWindowEnd.month - 1, rules.kannWindowEnd.day);
  return sixthBirthday > windowStart && sixthBirthday <= windowEnd ? kannYear : null;
}
