import { calculateMonthlySalary } from '../salary';
import type { EmployeeContract, PayPlanDetail } from '@/lib/api/types';

// Fix "today" so isActivePeriod is deterministic
beforeAll(() => {
  jest.useFakeTimers({ now: new Date('2026-03-08T12:00:00Z') });
});
afterAll(() => {
  jest.useRealTimers();
});

const contract: EmployeeContract = {
  id: 1,
  employee_id: 1,
  version: 1,
  from: '2025-01-01',
  to: '',
  section_id: 1,
  section_name: '',
  staff_category: 'educator',
  grade: 'S8a',
  step: 3,
  weekly_hours: 30,
  payplan_id: 1,
  payplan_name: '',
  properties: {},
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
};

const payPlan: PayPlanDetail = {
  id: 1,
  organization_id: 1,
  name: 'TVöD-SuE',
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
  periods: [
    {
      id: 1,
      payplan_id: 1,
      from: '2025-01-01',
      to: '',
      weekly_hours: 39,
      employer_contribution_rate: 2200,
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
      entries: [
        {
          id: 1,
          period_id: 1,
          grade: 'S8a',
          step: 3,
          step_min_years: 0,
          monthly_amount: 350000, // 3500.00 EUR in cents
          created_at: '2025-01-01T00:00:00Z',
          updated_at: '2025-01-01T00:00:00Z',
        },
      ],
    },
  ],
};

describe('calculateMonthlySalary', () => {
  it('calculates prorated salary based on hours', () => {
    const result = calculateMonthlySalary(contract, payPlan);
    // 350000 * (30 / 39) = 269230.77 → rounded to 269231
    expect(result).toBe(Math.round(350000 * (30 / 39)));
  });

  it('returns null when no active period', () => {
    const expiredPlan: PayPlanDetail = {
      ...payPlan,
      periods: [
        {
          ...payPlan.periods[0]!,
          from: '2020-01-01',
          to: '2020-12-31',
        },
      ],
    };
    expect(calculateMonthlySalary(contract, expiredPlan)).toBeNull();
  });

  it('returns null when no periods', () => {
    expect(calculateMonthlySalary(contract, { ...payPlan, periods: [] })).toBeNull();
    // Generated types require periods to be present, but the runtime may still
    // pass undefined; cast to exercise the defensive null-check.
    expect(
      calculateMonthlySalary(contract, {
        ...payPlan,
        periods: undefined as unknown as PayPlanDetail['periods'],
      })
    ).toBeNull();
  });

  it('returns null when no matching entry for grade/step', () => {
    const mismatchContract = { ...contract, grade: 'S11b', step: 6 };
    expect(calculateMonthlySalary(mismatchContract, payPlan)).toBeNull();
  });

  it('returns null when weekly_hours is 0', () => {
    const zeroPlan: PayPlanDetail = {
      ...payPlan,
      periods: [
        {
          ...payPlan.periods[0]!,
          weekly_hours: 0,
        },
      ],
    };
    expect(calculateMonthlySalary(contract, zeroPlan)).toBeNull();
  });

  it('returns full amount when contract hours equal plan hours', () => {
    const fullTimeContract = { ...contract, weekly_hours: 39 };
    expect(calculateMonthlySalary(fullTimeContract, payPlan)).toBe(350000);
  });
});
