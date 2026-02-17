import { z } from 'zod';
import { isDateBefore } from '@/lib/utils/contracts';

export const budgetItemSchema = z.object({
  name: z.string().min(1).max(255),
  category: z.enum(['income', 'expense']),
  per_child: z.boolean(),
});

export const budgetItemEntrySchema = z
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

// Combined schema for creating a budget item with an initial entry
export const budgetItemWithEntrySchema = budgetItemSchema
  .extend({
    entry_from: z.string().min(1),
    entry_to: z.string().optional(),
    entry_amount_euros: z.number().min(0),
    entry_notes: z.string().max(500).optional(),
  })
  .refine((data) => !data.entry_to || !isDateBefore(data.entry_to, data.entry_from), {
    path: ['entry_to'],
    message: 'End date must be after start date',
  });

export type BudgetItemFormData = z.infer<typeof budgetItemSchema>;
export type BudgetItemEntryFormData = z.infer<typeof budgetItemEntrySchema>;
export type BudgetItemWithEntryFormData = z.infer<typeof budgetItemWithEntrySchema>;
