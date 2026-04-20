/**
 * School-enrollment (Einschulung) rules per German Bundesland.
 *
 * Berlin rule (https://familienportal.berlin.de/artikel/einschulung-frueher-oder-spaeter):
 *   - Muss-Kind: turns 6 on or before Sep 30 → school starts that August.
 *   - Kann-Kind: turns 6 between Oct 1 and Mar 31 of the following calendar
 *     year → parents may apply for early enrollment the preceding August.
 */
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

export type SchoolEnrollment = {
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
  state: string
): SchoolEnrollment | null {
  if (!birthdate || !state) return null;
  const bd = parseBirthdate(birthdate);
  if (!bd) return null;
  const rules = rulesByState[state];
  if (!rules) return null;

  const mussYear = computeMussYear(bd, rules.stichtag);
  const kannYear = computeKannYear(bd, rules, mussYear);

  return {
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
