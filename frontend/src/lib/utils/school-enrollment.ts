/**
 * Stichtag (enrollment cutoff) dates by German state.
 * Children who turn 6 on or before this date start school that year.
 */
const stichtagByState: Record<string, { month: number; day: number }> = {
  berlin: { month: 9, day: 30 }, // September 30
};

const defaultStichtag = { month: 9, day: 30 };

/**
 * Calculate the suggested Kita contract end date based on the child's birthdate
 * and the organization's state (Bundesland).
 *
 * Rule: If the child turns 6 on or before the state's Stichtag (cutoff date),
 * they start school that year. Otherwise, they start the following year.
 * School starts in August, so the Kita contract ends July 31.
 *
 * @param birthdate - Child's birthdate in YYYY-MM-DD format
 * @param state - Organization's state (e.g., "berlin")
 * @returns Contract end date in YYYY-MM-DD format, or null if inputs are invalid
 */
export function calculateContractEndDate(birthdate: string, state: string): string | null {
  if (!birthdate || !state) return null;

  const bd = new Date(birthdate + 'T00:00:00');
  if (isNaN(bd.getTime())) return null;

  const stichtag = stichtagByState[state] || defaultStichtag;

  // Year when the child turns 6
  const turnsSixYear = bd.getFullYear() + 6;

  // Does the child turn 6 on or before the Stichtag in that year?
  const birthdayInTurnsSixYear = new Date(turnsSixYear, bd.getMonth(), bd.getDate());
  const stichtagDate = new Date(turnsSixYear, stichtag.month - 1, stichtag.day);

  const schoolStartYear = birthdayInTurnsSixYear <= stichtagDate ? turnsSixYear : turnsSixYear + 1;

  // Kita contract ends July 31 of the school start year
  return `${schoolStartYear}-07-31`;
}
