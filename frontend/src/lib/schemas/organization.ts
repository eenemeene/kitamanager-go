import { z } from 'zod';
import type { OrganizationCreateRequest, OrganizationUpdateRequest } from '@/lib/api/types';

const organizationBaseSchema = z.object({
  name: z.string().min(1).max(255),
  state: z.string().min(1),
  active: z.boolean().default(true),
});

export const organizationCreateSchema = organizationBaseSchema.extend({
  default_section_name: z.string().min(1).max(255),
}) satisfies z.ZodType<OrganizationCreateRequest>;

export const organizationUpdateSchema =
  organizationBaseSchema satisfies z.ZodType<OrganizationUpdateRequest>;

// Use the create schema as the default form schema (superset of fields)
export const organizationSchema = organizationBaseSchema.extend({
  default_section_name: z.string().min(1).max(255).optional(),
});

export type OrganizationFormData = z.infer<typeof organizationSchema>;
export type OrganizationCreateFormData = z.infer<typeof organizationCreateSchema>;
export type OrganizationUpdateFormData = z.infer<typeof organizationUpdateSchema>;
