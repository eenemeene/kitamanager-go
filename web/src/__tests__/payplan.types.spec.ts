import { describe, it, expect } from 'vitest'
import type {
  PayPlan,
  PayPlanPeriod,
  PayPlanEntry,
  PayPlanCreateRequest,
  PayPlanUpdateRequest,
  PayPlanPeriodCreateRequest,
  PayPlanPeriodUpdateRequest,
  PayPlanEntryCreateRequest,
  PayPlanEntryUpdateRequest
} from '../api/types'

describe('PayPlan Types', () => {
  describe('PayPlan', () => {
    it('should allow valid PayPlan object', () => {
      const payPlan: PayPlan = {
        id: 1,
        organization_id: 1,
        name: 'TVöD 2024',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z'
      }

      expect(payPlan.id).toBe(1)
      expect(payPlan.name).toBe('TVöD 2024')
      expect(payPlan.organization_id).toBe(1)
    })

    it('should allow PayPlan with optional periods', () => {
      const payPlan: PayPlan = {
        id: 1,
        organization_id: 1,
        name: 'TVöD 2024',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        periods: [],
        total_periods: 0
      }

      expect(payPlan.periods).toEqual([])
      expect(payPlan.total_periods).toBe(0)
    })

    it('should allow PayPlan with periods containing entries', () => {
      const entry: PayPlanEntry = {
        id: 1,
        period_id: 1,
        grade: 'S8a',
        step: 3,
        monthly_amount: 350000,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z'
      }

      const period: PayPlanPeriod = {
        id: 1,
        payplan_id: 1,
        from: '2024-01-01',
        to: null,
        weekly_hours: 39.0,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        entries: [entry]
      }

      const payPlan: PayPlan = {
        id: 1,
        organization_id: 1,
        name: 'TVöD 2024',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        periods: [period],
        total_periods: 1
      }

      expect(payPlan.periods).toHaveLength(1)
      expect(payPlan.periods![0].entries).toHaveLength(1)
      expect(payPlan.periods![0].entries![0].grade).toBe('S8a')
      expect(payPlan.periods![0].entries![0].monthly_amount).toBe(350000)
    })
  })

  describe('PayPlanPeriod', () => {
    it('should allow valid PayPlanPeriod with null end date (ongoing)', () => {
      const period: PayPlanPeriod = {
        id: 1,
        payplan_id: 1,
        from: '2024-01-01',
        to: null,
        weekly_hours: 39.0,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z'
      }

      expect(period.from).toBe('2024-01-01')
      expect(period.to).toBeNull()
      expect(period.weekly_hours).toBe(39.0)
    })

    it('should allow PayPlanPeriod with defined end date', () => {
      const period: PayPlanPeriod = {
        id: 1,
        payplan_id: 1,
        from: '2024-01-01',
        to: '2024-12-31',
        weekly_hours: 39.0,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z'
      }

      expect(period.to).toBe('2024-12-31')
    })

    it('should allow PayPlanPeriod with different weekly hours values', () => {
      const fullTimePeriod: PayPlanPeriod = {
        id: 1,
        payplan_id: 1,
        from: '2024-01-01',
        to: null,
        weekly_hours: 40.0,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z'
      }

      const partTimePeriod: PayPlanPeriod = {
        id: 2,
        payplan_id: 1,
        from: '2024-01-01',
        to: null,
        weekly_hours: 20.0,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z'
      }

      expect(fullTimePeriod.weekly_hours).toBe(40.0)
      expect(partTimePeriod.weekly_hours).toBe(20.0)
    })
  })

  describe('PayPlanEntry', () => {
    it('should allow valid PayPlanEntry object', () => {
      const entry: PayPlanEntry = {
        id: 1,
        period_id: 1,
        grade: 'S8a',
        step: 3,
        monthly_amount: 350000,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z'
      }

      expect(entry.grade).toBe('S8a')
      expect(entry.step).toBe(3)
      expect(entry.monthly_amount).toBe(350000)
    })

    it('should allow PayPlanEntry with different grades', () => {
      const grades = ['S3', 'S8a', 'S8b', 'S11b', 'S15']

      grades.forEach((grade, index) => {
        const entry: PayPlanEntry = {
          id: index + 1,
          period_id: 1,
          grade,
          step: 1,
          monthly_amount: 300000 + index * 10000,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        }
        expect(entry.grade).toBe(grade)
      })
    })

    it('should allow PayPlanEntry with steps 1-6', () => {
      for (let step = 1; step <= 6; step++) {
        const entry: PayPlanEntry = {
          id: step,
          period_id: 1,
          grade: 'S8a',
          step,
          monthly_amount: 300000 + step * 20000,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        }
        expect(entry.step).toBe(step)
      }
    })
  })

  describe('Request Types', () => {
    it('should allow valid PayPlanCreateRequest', () => {
      const request: PayPlanCreateRequest = {
        name: 'TVöD 2024'
      }

      expect(request.name).toBe('TVöD 2024')
    })

    it('should allow valid PayPlanUpdateRequest with optional fields', () => {
      const request: PayPlanUpdateRequest = {
        name: 'TVöD 2025 Updated'
      }

      expect(request.name).toBe('TVöD 2025 Updated')
    })

    it('should allow valid PayPlanPeriodCreateRequest', () => {
      const request: PayPlanPeriodCreateRequest = {
        from: '2024-01-01',
        to: '2024-12-31',
        weekly_hours: 39.0
      }

      expect(request.from).toBe('2024-01-01')
      expect(request.to).toBe('2024-12-31')
      expect(request.weekly_hours).toBe(39.0)
    })

    it('should allow PayPlanPeriodCreateRequest with null end date', () => {
      const request: PayPlanPeriodCreateRequest = {
        from: '2024-01-01',
        to: null,
        weekly_hours: 39.0
      }

      expect(request.to).toBeNull()
    })

    it('should allow valid PayPlanPeriodUpdateRequest with partial fields', () => {
      const request: PayPlanPeriodUpdateRequest = {
        weekly_hours: 40.0
      }

      expect(request.weekly_hours).toBe(40.0)
      expect(request.from).toBeUndefined()
      expect(request.to).toBeUndefined()
    })

    it('should allow valid PayPlanEntryCreateRequest', () => {
      const request: PayPlanEntryCreateRequest = {
        grade: 'S8a',
        step: 3,
        monthly_amount: 350000
      }

      expect(request.grade).toBe('S8a')
      expect(request.step).toBe(3)
      expect(request.monthly_amount).toBe(350000)
    })

    it('should allow valid PayPlanEntryUpdateRequest with partial fields', () => {
      const request: PayPlanEntryUpdateRequest = {
        monthly_amount: 360000
      }

      expect(request.monthly_amount).toBe(360000)
      expect(request.grade).toBeUndefined()
      expect(request.step).toBeUndefined()
    })
  })

  describe('Real-world scenarios', () => {
    it('should represent a complete TVöD pay scale structure', () => {
      // Create entries for S8a steps 1-6
      const entries: PayPlanEntry[] = [
        {
          id: 1,
          period_id: 1,
          grade: 'S8a',
          step: 1,
          monthly_amount: 314847,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        },
        {
          id: 2,
          period_id: 1,
          grade: 'S8a',
          step: 2,
          monthly_amount: 329947,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        },
        {
          id: 3,
          period_id: 1,
          grade: 'S8a',
          step: 3,
          monthly_amount: 350089,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        },
        {
          id: 4,
          period_id: 1,
          grade: 'S8a',
          step: 4,
          monthly_amount: 377571,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        },
        {
          id: 5,
          period_id: 1,
          grade: 'S8a',
          step: 5,
          monthly_amount: 405054,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        },
        {
          id: 6,
          period_id: 1,
          grade: 'S8a',
          step: 6,
          monthly_amount: 423251,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        }
      ]

      const period: PayPlanPeriod = {
        id: 1,
        payplan_id: 1,
        from: '2024-03-01',
        to: null,
        weekly_hours: 39.0,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        entries
      }

      const payPlan: PayPlan = {
        id: 1,
        organization_id: 1,
        name: 'TVöD-SuE 2024',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        periods: [period],
        total_periods: 1
      }

      // Verify structure
      expect(payPlan.name).toBe('TVöD-SuE 2024')
      expect(payPlan.periods![0].weekly_hours).toBe(39.0)
      expect(payPlan.periods![0].entries!).toHaveLength(6)

      // Verify salary progression (each step should be higher than previous)
      const amounts = payPlan.periods![0].entries!.map((e) => e.monthly_amount)
      for (let i = 1; i < amounts.length; i++) {
        expect(amounts[i]).toBeGreaterThan(amounts[i - 1])
      }
    })

    it('should support multiple pay plans per organization', () => {
      const payPlans: PayPlan[] = [
        {
          id: 1,
          organization_id: 1,
          name: 'TVöD-SuE',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        },
        {
          id: 2,
          organization_id: 1,
          name: 'TVöD-VKA',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        },
        {
          id: 3,
          organization_id: 1,
          name: 'Custom Pay Plan',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        }
      ]

      expect(payPlans).toHaveLength(3)
      expect(payPlans.every((p) => p.organization_id === 1)).toBe(true)
    })

    it('should support historical periods', () => {
      const periods: PayPlanPeriod[] = [
        {
          id: 1,
          payplan_id: 1,
          from: '2022-01-01',
          to: '2022-12-31',
          weekly_hours: 39.0,
          created_at: '2022-01-01T00:00:00Z',
          updated_at: '2022-01-01T00:00:00Z'
        },
        {
          id: 2,
          payplan_id: 1,
          from: '2023-01-01',
          to: '2023-12-31',
          weekly_hours: 39.0,
          created_at: '2023-01-01T00:00:00Z',
          updated_at: '2023-01-01T00:00:00Z'
        },
        {
          id: 3,
          payplan_id: 1,
          from: '2024-01-01',
          to: null,
          weekly_hours: 39.0,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z'
        }
      ]

      // Verify historical periods have end dates
      expect(periods[0].to).toBe('2022-12-31')
      expect(periods[1].to).toBe('2023-12-31')
      // Current period is ongoing
      expect(periods[2].to).toBeNull()
    })
  })
})
