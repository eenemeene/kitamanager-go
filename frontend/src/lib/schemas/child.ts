import { z } from 'zod';
import { isDateBefore } from '@/lib/utils/contracts';

export const childSchema = z.object({
  first_name: z.string().min(1),
  last_name: z.string().min(1),
  gender: z.enum(['male', 'female', 'diverse']),
  birthdate: z.string().min(1),
});

export const childContractSchema = z
  .object({
    from: z.string().min(1),
    to: z.string().optional(),
    properties: z.record(z.string()).optional(),
  })
  .refine((data) => !data.to || !isDateBefore(data.to, data.from), {
    path: ['to'],
    message: 'End date must be after start date',
  });

// Combined schema for creating a child with an initial contract
export const childWithContractSchema = z
  .object({
    first_name: z.string().min(1),
    last_name: z.string().min(1),
    gender: z.enum(['male', 'female', 'diverse']),
    birthdate: z.string().min(1),
    contract_from: z.string().min(1),
    contract_to: z.string().optional(),
    properties: z.record(z.string()).optional(),
  })
  .refine((data) => !data.contract_to || !isDateBefore(data.contract_to, data.contract_from), {
    path: ['contract_to'],
    message: 'End date must be after start date',
  })
  .refine((data) => !isDateBefore(data.contract_from, data.birthdate), {
    path: ['contract_from'],
    message: 'Contract start date cannot be before birthdate',
  });

export type ChildFormData = z.infer<typeof childSchema>;
export type ChildContractFormData = z.infer<typeof childContractSchema>;
export type ChildWithContractFormData = z.infer<typeof childWithContractSchema>;
