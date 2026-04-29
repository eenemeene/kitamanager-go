import { z } from 'zod';
import type { BudgetItemCreateRequest } from '@/lib/api/types';
import { endDateAfterStart } from './period';

export const budgetItemSchema = z.object({
  name: z.string().min(1).max(255),
  category: z.enum(['income', 'expense']),
  per_child: z.boolean(),
}) satisfies z.ZodType<BudgetItemCreateRequest>;

// budgetItemEntrySchema is form-only: API stores `amount_cents` (int),
// the form collects `amount_euros`. Conversion happens in the submit
// handler.
export const budgetItemEntrySchema = z
  .object({
    from: z.string().min(1),
    to: z.string().optional(),
    amount_euros: z.number().min(0),
    notes: z.string().max(500).optional(),
  })
  .refine(...endDateAfterStart('from', 'to'));

// Combined schema for creating a budget item with an initial entry —
// composite shape with no single API counterpart (the page splits it
// into BudgetItemCreateRequest + BudgetItemEntryCreateRequest before
// submitting). Skipping the satisfies guard.
export const budgetItemWithEntrySchema = budgetItemSchema
  .extend({
    entry_from: z.string().min(1),
    entry_to: z.string().optional(),
    entry_amount_euros: z.number().min(0),
    entry_notes: z.string().max(500).optional(),
  })
  .refine(...endDateAfterStart('entry_from', 'entry_to'));

export type BudgetItemFormData = z.infer<typeof budgetItemSchema>;
export type BudgetItemEntryFormData = z.infer<typeof budgetItemEntrySchema>;
export type BudgetItemWithEntryFormData = z.infer<typeof budgetItemWithEntrySchema>;
