import { z } from 'zod';

export const employeeSchema = z.object({
  first_name: z.string().min(1).max(255),
  last_name: z.string().min(1).max(255),
  gender: z.enum(['male', 'female', 'diverse']),
  birthdate: z.string().min(1),
});

export const employeeContractSchema = z
  .object({
    from: z.string().min(1),
    to: z.string().optional(),
    payplan_id: z.number().min(1),
    staff_category: z.enum(['qualified', 'supplementary', 'non_pedagogical']),
    grade: z.string().min(1).max(20),
    step: z.number().min(0).max(10),
    weekly_hours: z.number().min(0).max(168),
  })
  .refine((data) => !data.to || data.to >= data.from, {
    path: ['to'],
    message: 'End date must be after start date',
  });

export type EmployeeFormData = z.infer<typeof employeeSchema>;
export type EmployeeContractFormData = z.infer<typeof employeeContractSchema>;
