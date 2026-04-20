import { z } from 'zod';

export const passwordSchema = z.string().min(8).max(72);

export const userCreateSchema = z.object({
  name: z.string().min(1).max(255),
  email: z.string().email(),
  password: passwordSchema,
  active: z.boolean().default(true),
});

export const userUpdateSchema = z.object({
  name: z.string().min(1).max(255),
  email: z.string().email(),
  active: z.boolean().default(true),
});

export const changePasswordSchema = z
  .object({
    current_password: z.string().min(1),
    new_password: passwordSchema,
    confirm_password: z.string(),
  })
  .refine((d) => d.new_password === d.confirm_password, {
    path: ['confirm_password'],
    message: 'settings.password.validation.mismatch',
  })
  .refine((d) => d.new_password !== d.current_password, {
    path: ['new_password'],
    message: 'settings.password.validation.sameAsCurrent',
  });

export type UserCreateFormData = z.infer<typeof userCreateSchema>;
export type UserUpdateFormData = z.infer<typeof userUpdateSchema>;
export type ChangePasswordFormData = z.infer<typeof changePasswordSchema>;
