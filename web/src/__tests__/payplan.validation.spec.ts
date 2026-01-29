import { describe, it, expect } from 'vitest'
import { payPlanSchema, payPlanPeriodSchema, payPlanEntrySchema } from '../validation/schemas'

describe('PayPlan Validation Schemas', () => {
  describe('payPlanSchema', () => {
    it('should accept valid pay plan data', () => {
      const data = { name: 'TVöD 2024' }
      const result = payPlanSchema.safeParse(data)

      expect(result.success).toBe(true)
      if (result.success) {
        expect(result.data.name).toBe('TVöD 2024')
      }
    })

    it('should trim whitespace from name', () => {
      const data = { name: '  TVöD 2024  ' }
      const result = payPlanSchema.safeParse(data)

      expect(result.success).toBe(true)
      if (result.success) {
        expect(result.data.name).toBe('TVöD 2024')
      }
    })

    it('should reject empty name', () => {
      const data = { name: '' }
      const result = payPlanSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject whitespace-only name', () => {
      const data = { name: '   ' }
      const result = payPlanSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject missing name', () => {
      const data = {}
      const result = payPlanSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject name exceeding max length', () => {
      const data = { name: 'a'.repeat(256) }
      const result = payPlanSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should accept name at max length', () => {
      const data = { name: 'a'.repeat(255) }
      const result = payPlanSchema.safeParse(data)

      expect(result.success).toBe(true)
    })
  })

  describe('payPlanPeriodSchema', () => {
    it('should accept valid period data with no end date', () => {
      const data = {
        from_date: new Date('2024-01-01'),
        to_date: null,
        weekly_hours: 39.0
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(true)
      if (result.success) {
        expect(result.data.weekly_hours).toBe(39.0)
      }
    })

    it('should accept valid period data with end date', () => {
      const data = {
        from_date: new Date('2024-01-01'),
        to_date: new Date('2024-12-31'),
        weekly_hours: 39.0
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(true)
    })

    it('should reject missing from_date', () => {
      const data = {
        to_date: null,
        weekly_hours: 39.0
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject missing weekly_hours', () => {
      const data = {
        from_date: new Date('2024-01-01'),
        to_date: null
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject weekly_hours of 0', () => {
      const data = {
        from_date: new Date('2024-01-01'),
        to_date: null,
        weekly_hours: 0
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject negative weekly_hours', () => {
      const data = {
        from_date: new Date('2024-01-01'),
        to_date: null,
        weekly_hours: -10
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject weekly_hours exceeding 168', () => {
      const data = {
        from_date: new Date('2024-01-01'),
        to_date: null,
        weekly_hours: 169
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should accept various valid weekly_hours values', () => {
      const validHours = [0.1, 20, 30.5, 39, 40, 50, 168]

      validHours.forEach((hours) => {
        const data = {
          from_date: new Date('2024-01-01'),
          to_date: null,
          weekly_hours: hours
        }
        const result = payPlanPeriodSchema.safeParse(data)
        expect(result.success).toBe(true)
      })
    })

    it('should reject to_date before from_date', () => {
      const data = {
        from_date: new Date('2024-12-31'),
        to_date: new Date('2024-01-01'),
        weekly_hours: 39.0
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should accept to_date equal to from_date', () => {
      const data = {
        from_date: new Date('2024-01-01'),
        to_date: new Date('2024-01-01'),
        weekly_hours: 39.0
      }
      const result = payPlanPeriodSchema.safeParse(data)

      expect(result.success).toBe(true)
    })
  })

  describe('payPlanEntrySchema', () => {
    it('should accept valid entry data', () => {
      const data = {
        grade: 'S8a',
        step: 3,
        monthly_amount: 350000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(true)
      if (result.success) {
        expect(result.data.grade).toBe('S8a')
        expect(result.data.step).toBe(3)
        expect(result.data.monthly_amount).toBe(350000)
      }
    })

    it('should trim whitespace from grade', () => {
      const data = {
        grade: '  S8a  ',
        step: 1,
        monthly_amount: 300000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(true)
      if (result.success) {
        expect(result.data.grade).toBe('S8a')
      }
    })

    it('should reject empty grade', () => {
      const data = {
        grade: '',
        step: 1,
        monthly_amount: 300000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject whitespace-only grade', () => {
      const data = {
        grade: '   ',
        step: 1,
        monthly_amount: 300000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject missing grade', () => {
      const data = {
        step: 1,
        monthly_amount: 300000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should accept steps 1-6', () => {
      for (let step = 1; step <= 6; step++) {
        const data = {
          grade: 'S8a',
          step,
          monthly_amount: 300000
        }
        const result = payPlanEntrySchema.safeParse(data)
        expect(result.success).toBe(true)
      }
    })

    it('should reject step 0', () => {
      const data = {
        grade: 'S8a',
        step: 0,
        monthly_amount: 300000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject step 7', () => {
      const data = {
        grade: 'S8a',
        step: 7,
        monthly_amount: 300000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject negative step', () => {
      const data = {
        grade: 'S8a',
        step: -1,
        monthly_amount: 300000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should reject missing step', () => {
      const data = {
        grade: 'S8a',
        monthly_amount: 300000
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should accept monthly_amount of 0', () => {
      const data = {
        grade: 'S8a',
        step: 1,
        monthly_amount: 0
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(true)
    })

    it('should reject negative monthly_amount', () => {
      const data = {
        grade: 'S8a',
        step: 1,
        monthly_amount: -100
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should accept large monthly_amount (in cents)', () => {
      const data = {
        grade: 'S8a',
        step: 6,
        monthly_amount: 1000000 // €10,000.00
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(true)
    })

    it('should reject missing monthly_amount', () => {
      const data = {
        grade: 'S8a',
        step: 1
      }
      const result = payPlanEntrySchema.safeParse(data)

      expect(result.success).toBe(false)
    })

    it('should accept various grade formats', () => {
      const grades = ['S3', 'S8a', 'S8b', 'S11b', 'S15', 'E9', 'E13', 'A13']

      grades.forEach((grade) => {
        const data = {
          grade,
          step: 1,
          monthly_amount: 300000
        }
        const result = payPlanEntrySchema.safeParse(data)
        expect(result.success).toBe(true)
      })
    })
  })
})
