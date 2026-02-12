import { z } from 'zod';

export const userCreateSchema = z.object({
  name: z.string().min(1).max(255),
  email: z.string().email(),
  password: z.string().min(8).max(72),
  active: z.boolean().default(true),
});

export const userUpdateSchema = z.object({
  name: z.string().min(1).max(255),
  email: z.string().email(),
  active: z.boolean().default(true),
});

export type UserCreateFormData = z.infer<typeof userCreateSchema>;
export type UserUpdateFormData = z.infer<typeof userUpdateSchema>;
