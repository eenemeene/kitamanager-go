import { z } from 'zod';

const optionalAge = z.preprocess((val) => {
  if (val === '' || val === null || val === undefined) return null;
  const num = Number(val);
  return isNaN(num) ? null : num;
}, z.number().int().min(0).nullable());

export const sectionSchema = z
  .object({
    name: z.string().min(1).max(255),
    min_age_months: optionalAge.optional(),
    max_age_months: optionalAge.optional(),
  })
  .refine(
    (data) => {
      if (data.min_age_months != null && data.max_age_months != null) {
        return data.min_age_months < data.max_age_months;
      }
      return true;
    },
    {
      message: 'min_age_months must be less than max_age_months',
      path: ['max_age_months'],
    }
  );

export type SectionFormData = z.infer<typeof sectionSchema>;
