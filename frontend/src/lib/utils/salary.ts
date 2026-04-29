import type { EmployeeContract, PayPlanDetail } from '@/lib/api/types';
import { isActivePeriod } from '@/lib/utils/contracts';

export function calculateMonthlySalary(
  contract: EmployeeContract,
  payPlan: PayPlanDetail
): number | null {
  const period = payPlan.periods?.find((p) => isActivePeriod(p));
  if (!period) return null;

  const entry = period.entries?.find((e) => e.grade === contract.grade && e.step === contract.step);
  if (!entry) return null;

  if (!period.weekly_hours) return null;
  return Math.round(entry.monthly_amount * (contract.weekly_hours / period.weekly_hours));
}
