import { z } from 'zod';
import { isDateBefore } from '@/lib/utils/contracts';

export const governmentFundingSchema = z.object({
  name: z.string().min(1).max(255),
  state: z.string().min(1),
});

export const governmentFundingPeriodSchema = z
  .object({
    from: z.string().min(1),
    to: z.string().optional(),
    comment: z.string().max(1000).optional(),
  })
  .refine((data) => !data.to || !isDateBefore(data.to, data.from), {
    path: ['to'],
    message: 'End date must be after start date',
  });

export const governmentFundingPropertySchema = z.object({
  key: z.string().min(1).max(100),
  value: z.string().min(1).max(255),
  payment: z.number().min(0),
  requirement: z.number().min(0),
  min_age: z.number().min(0).optional().nullable(),
  max_age: z.number().min(0).optional().nullable(),
  comment: z.string().max(500).optional(),
});

export type GovernmentFundingFormData = z.infer<typeof governmentFundingSchema>;
export type GovernmentFundingPeriodFormData = z.infer<typeof governmentFundingPeriodSchema>;
export type GovernmentFundingPropertyFormData = z.infer<typeof governmentFundingPropertySchema>;
