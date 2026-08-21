import { todayBerlin } from './contracts';

/**
 * Calculate years of service from contract history.
 * Uses the earliest contract start date.
 */
export function calculateYearsOfService(
  contracts: { from: string }[],
  // Berlin's today by default: the eligible step this feeds is compared against
  // what the server computed, and an off-by-one day at a step boundary is a
  // disagreement about somebody's pay grade.
  //
  // The UTC-midnight form, not `todayBerlinDate()`. The contract dates below go
  // through `new Date(c.from)`, which reads a bare "YYYY-MM-DD" as UTC midnight,
  // so an `asOf` at *local* midnight would compare two different frames and put
  // a few hours of service on a contract that starts today.
  asOf: Date = new Date(todayBerlin())
): number {
  if (contracts.length === 0) return 0;

  const earliest = contracts.reduce((min, c) => {
    const d = new Date(c.from);
    return d < min ? d : min;
  }, new Date(contracts[0].from));

  const diffMs = asOf.getTime() - earliest.getTime();
  if (diffMs <= 0) return 0;

  return diffMs / (365.25 * 24 * 60 * 60 * 1000);
}

/**
 * Determine the eligible step based on years of service and pay plan entries.
 * Only considers entries that have step_min_years defined.
 * Returns 0 if no entries have step rules.
 */
export function determineEligibleStep(
  yearsOfService: number,
  entries: { step: number; grade: string; step_min_years?: number | null }[],
  grade: string
): number {
  const eligible = entries.filter(
    (e) => e.grade === grade && e.step_min_years != null && e.step_min_years <= yearsOfService
  );

  if (eligible.length === 0) return 0;

  return Math.max(...eligible.map((e) => e.step));
}
