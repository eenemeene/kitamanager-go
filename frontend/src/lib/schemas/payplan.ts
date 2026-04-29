import { z } from 'zod';
import type { PayPlanCreateRequest } from '@/lib/api/types';
import { endDateAfterStart } from './period';

export const payPlanSchema = z.object({
  name: z.string().min(1).max(255),
}) satisfies z.ZodType<PayPlanCreateRequest>;

// payPlanPeriodSchema is form-only: the API stores
// `employer_contribution_rate` in hundredths of a percent (2200 = 22.00%)
// but the form collects 0-100 percent. Conversion happens in the
// submit handler. Skipping the satisfies guard.
export const payPlanPeriodSchema = z
  .object({
    from: z.string().min(1),
    to: z.string().optional(),
    weekly_hours: z.number().gt(0).max(168),
    employer_contribution_rate: z.number().min(0).max(100),
  })
  .refine(...endDateAfterStart('from', 'to'));

// payPlanEntrySchema is form-only: API stores `monthly_amount` in cents,
// the form collects euros. Conversion happens in the submit handler.
export const payPlanEntrySchema = z.object({
  grade: z.string().min(1),
  step: z.number().min(1).max(10),
  monthly_amount_euros: z.number().min(0),
  step_min_years: z.number().min(0).optional(),
});

export type PayPlanFormData = z.infer<typeof payPlanSchema>;
export type PayPlanPeriodFormData = z.infer<typeof payPlanPeriodSchema>;
export type PayPlanEntryFormData = z.infer<typeof payPlanEntrySchema>;
