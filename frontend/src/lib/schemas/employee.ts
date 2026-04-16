import { z } from 'zod';
import { personBaseSchema } from './person';
import { endDateAfterStart } from './period';

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
  .refine(...endDateAfterStart('from', 'to'));

export type EmployeeFormData = z.infer<typeof employeeSchema>;
export type EmployeeContractFormData = z.infer<typeof employeeContractSchema>;
