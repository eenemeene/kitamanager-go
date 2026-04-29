import { z } from 'zod';
import type { EmployeeContractCreateRequest } from '@/lib/api/types';
import { personBaseSchema } from './person';
import { endDateAfterStart } from './period';

// employeeSchema reuses personBaseSchema. There is no top-level Person
// schema in the API (the spec inlines person fields into Child/Employee
// concrete types) so no satisfies guard is appropriate here.
export const employeeSchema = personBaseSchema;

export const employeeContractSchema = z
  .object({
    from: z.string().min(1),
    to: z.string().optional(),
    section_id: z.number().min(1, 'Section is required'),
    payplan_id: z.number().min(1),
    staff_category: z.enum(['qualified', 'supplementary', 'non_pedagogical']),
    grade: z.string().min(1).max(20),
    step: z.number().min(1).max(10),
    weekly_hours: z.number().min(0).max(168),
  })
  .refine(...endDateAfterStart('from', 'to')) satisfies z.ZodType<EmployeeContractCreateRequest>;

export type EmployeeFormData = z.infer<typeof employeeSchema>;
export type EmployeeContractFormData = z.infer<typeof employeeContractSchema>;
