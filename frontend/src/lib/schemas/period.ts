import { isDateBefore } from '@/lib/utils/contracts';

/**
 * Zod `.refine()` arguments for validating that an optional end date
 * is not before a required start date.
 *
 * Usage:
 *   .refine(...endDateAfterStart('from', 'to'))
 */
export function endDateAfterStart(
  fromField: string,
  toField: string
): [check: (data: Record<string, unknown>) => boolean, opts: { path: string[]; message: string }] {
  return [
    (data) => !data[toField] || !isDateBefore(data[toField] as string, data[fromField] as string),
    { path: [toField], message: 'End date must be after start date' },
  ];
}
