import { z } from 'zod'

// Gender enum shared across multiple schemas
export const genderSchema = z.enum(['male', 'female', 'diverse'], {
  required_error: 'validation.genderRequired'
})

// Person schema (shared by child and employee)
const personBaseSchema = {
  first_name: z
    .string({ required_error: 'validation.firstNameRequired' })
    .min(1, 'validation.firstNameRequired')
    .transform((v) => v.trim()),
  last_name: z
    .string({ required_error: 'validation.lastNameRequired' })
    .min(1, 'validation.lastNameRequired')
    .transform((v) => v.trim()),
  gender: genderSchema,
  birthdate: z.date({ required_error: 'validation.birthdateRequired' })
}

// Child schema
export const childSchema = z.object(personBaseSchema)
export type ChildFormData = z.infer<typeof childSchema>

// Employee schema (same as child for now)
export const employeeSchema = z.object(personBaseSchema)
export type EmployeeFormData = z.infer<typeof employeeSchema>

// User schema
export const userSchema = z.object({
  name: z
    .string({ required_error: 'validation.nameRequired' })
    .min(1, 'validation.nameRequired')
    .transform((v) => v.trim()),
  email: z.string({ required_error: 'validation.emailRequired' }).email('validation.invalidEmail'),
  password: z.string().min(6, 'validation.passwordTooShort').optional().or(z.literal(''))
})
export type UserFormData = z.infer<typeof userSchema>

// Organization schema
export const organizationSchema = z.object({
  name: z
    .string({ required_error: 'validation.nameRequired' })
    .min(1, 'validation.nameRequired')
    .max(255, 'validation.nameTooLong')
    .transform((v) => v.trim()),
  state: z
    .string({ required_error: 'validation.stateRequired' })
    .min(1, 'validation.stateRequired'),
  active: z.boolean().default(true)
})
export type OrganizationFormData = z.infer<typeof organizationSchema>

// Group schema
export const groupSchema = z.object({
  name: z
    .string({ required_error: 'validation.nameRequired' })
    .min(1, 'validation.nameRequired')
    .max(255, 'validation.nameTooLong')
    .transform((v) => v.trim()),
  active: z.boolean().default(true)
})
export type GroupFormData = z.infer<typeof groupSchema>

// Child contract schema
export const childContractSchema = z
  .object({
    from_date: z.date({ required_error: 'validation.fromDateRequired' }),
    to_date: z.date().nullable().optional(),
    attributes: z.array(z.string()).default([])
  })
  .refine(
    (data) => {
      if (data.to_date && data.from_date) {
        return data.to_date >= data.from_date
      }
      return true
    },
    {
      message: 'validation.toDateMustBeAfterFromDate',
      path: ['to_date']
    }
  )
export type ChildContractFormData = z.infer<typeof childContractSchema>

// Employee contract schema
export const employeeContractSchema = z
  .object({
    from_date: z.date({ required_error: 'validation.fromDateRequired' }),
    to_date: z.date().nullable().optional(),
    position: z
      .string({ required_error: 'validation.positionRequired' })
      .min(1, 'validation.positionRequired'),
    weekly_hours: z
      .number({ required_error: 'validation.weeklyHoursRequired' })
      .min(0, 'validation.weeklyHoursMin')
      .max(168, 'validation.weeklyHoursMax'),
    salary: z.number({ required_error: 'validation.salaryRequired' }).min(0, 'validation.salaryMin')
  })
  .refine(
    (data) => {
      if (data.to_date && data.from_date) {
        return data.to_date >= data.from_date
      }
      return true
    },
    {
      message: 'validation.toDateMustBeAfterFromDate',
      path: ['to_date']
    }
  )
export type EmployeeContractFormData = z.infer<typeof employeeContractSchema>

// Government funding schema
export const governmentFundingSchema = z.object({
  name: z
    .string({ required_error: 'validation.nameRequired' })
    .min(1, 'validation.nameRequired')
    .max(255, 'validation.nameTooLong')
    .transform((v) => v.trim()),
  state: z.string({ required_error: 'validation.stateRequired' }).min(1, 'validation.stateRequired')
})
export type GovernmentFundingFormData = z.infer<typeof governmentFundingSchema>

// PayPlan schema
export const payPlanSchema = z.object({
  name: z
    .string({ required_error: 'validation.nameRequired' })
    .trim()
    .min(1, 'validation.nameRequired')
    .max(255, 'validation.nameTooLong')
})
export type PayPlanFormData = z.infer<typeof payPlanSchema>

// PayPlan period schema
export const payPlanPeriodSchema = z
  .object({
    from_date: z.date({ required_error: 'validation.fromDateRequired' }),
    to_date: z.date().nullable().optional(),
    weekly_hours: z
      .number({ required_error: 'payPlans.weeklyHoursRequired' })
      .min(0.1, 'payPlans.weeklyHoursMin')
      .max(168, 'validation.weeklyHoursMax')
  })
  .refine(
    (data) => {
      if (data.to_date && data.from_date) {
        return data.to_date >= data.from_date
      }
      return true
    },
    {
      message: 'validation.toDateMustBeAfterFromDate',
      path: ['to_date']
    }
  )
export type PayPlanPeriodFormData = z.infer<typeof payPlanPeriodSchema>

// PayPlan entry schema
export const payPlanEntrySchema = z.object({
  grade: z
    .string({ required_error: 'payPlans.gradeRequired' })
    .trim()
    .min(1, 'payPlans.gradeRequired'),
  step: z
    .number({ required_error: 'payPlans.stepRequired' })
    .min(1, 'payPlans.stepMin')
    .max(6, 'payPlans.stepMax'),
  monthly_amount: z
    .number({ required_error: 'payPlans.monthlyAmountRequired' })
    .min(0, 'payPlans.monthlyAmountMin')
})
export type PayPlanEntryFormData = z.infer<typeof payPlanEntrySchema>
