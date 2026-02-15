import { z } from 'zod';
import { isDateBefore } from '@/lib/utils/contracts';

export const costSchema = z.object({
  name: z.string().min(1).max(255),
});

export const costEntrySchema = z
  .object({
    from: z.string().min(1),
    to: z.string().optional(),
    amount_euros: z.number().min(0),
    notes: z.string().max(500).optional(),
  })
  .refine((data) => !data.to || !isDateBefore(data.to, data.from), {
    path: ['to'],
    message: 'End date must be after start date',
  });

export type CostFormData = z.infer<typeof costSchema>;
export type CostEntryFormData = z.infer<typeof costEntrySchema>;
