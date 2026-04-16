import { z } from 'zod';
import { endDateAfterStart } from './period';

export const payPlanSchema = z.object({
  name: z.string().min(1).max(255),
});

export const payPlanPeriodSchema = z
  .object({
    from: z.string().min(1),
    to: z.string().optional(),
    weekly_hours: z.number().gt(0).max(168),
    employer_contribution_rate: z.number().min(0).max(100),
  })
  .refine(...endDateAfterStart('from', 'to'));

export const payPlanEntrySchema = z.object({
  grade: z.string().min(1),
  step: z.number().min(1).max(10),
  monthly_amount_euros: z.number().min(0),
  step_min_years: z.number().min(0).optional(),
});

export type PayPlanFormData = z.infer<typeof payPlanSchema>;
export type PayPlanPeriodFormData = z.infer<typeof payPlanPeriodSchema>;
export type PayPlanEntryFormData = z.infer<typeof payPlanEntrySchema>;
