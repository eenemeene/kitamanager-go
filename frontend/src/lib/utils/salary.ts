import type { EmployeeContract, PayPlanDetail } from '@/lib/api/types';
import { isActivePeriod } from '@/lib/utils/contracts';

/**
 * The monthly salary a contract earns under a pay plan, in cents.
 *
 * `asOf` ("YYYY-MM-DD", default today in Europe/Berlin) selects the pay-plan
 * period. A roster shown for a past month has to price itself with the
 * Entgelttabelle that was in force then, not with the one in force now — the
 * two differ at every tariff round, which is the whole reason periods exist.
 */
export function calculateMonthlySalary(
  contract: EmployeeContract,
  payPlan: PayPlanDetail,
  asOf?: string | null
): number | null {
  const period = payPlan.periods?.find((p) => isActivePeriod(p, asOf));
  if (!period) return null;

  const entry = period.entries?.find((e) => e.grade === contract.grade && e.step === contract.step);
  if (!entry) return null;

  if (!period.weekly_hours) return null;
  return Math.round(entry.monthly_amount * (contract.weekly_hours / period.weekly_hours));
}
