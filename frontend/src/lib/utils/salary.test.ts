import { calculateMonthlySalary } from './salary';
import type { EmployeeContract, PayPlan } from '@/lib/api/types';

// Helper to create a base contract
function makeContract(overrides: Partial<EmployeeContract> = {}): EmployeeContract {
  return {
    id: 1,
    employee_id: 1,
    from: '2024-01-01',
    to: null,
    staff_category: 'qualified',
    grade: 'S8a',
    step: 3,
    weekly_hours: 39,
    payplan_id: 1,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
}

// Helper to create a base payplan
function makePayPlan(overrides: Partial<PayPlan> = {}): PayPlan {
  return {
    id: 1,
    organization_id: 1,
    name: 'TVöD SuE 2024',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    periods: [
      {
        id: 1,
        payplan_id: 1,
        from: '2024-01-01',
        to: null,
        weekly_hours: 39,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        entries: [
          {
            id: 1,
            period_id: 1,
            grade: 'S8a',
            step: 3,
            monthly_amount: 350000, // 3500.00 EUR
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
      },
    ],
    ...overrides,
  };
}

describe('calculateMonthlySalary', () => {
  // Fix the date for deterministic tests
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-06-15'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('returns correct full-time salary when contract matches period hours', () => {
    const contract = makeContract({ weekly_hours: 39 });
    const payPlan = makePayPlan();

    const result = calculateMonthlySalary(contract, payPlan);

    // 350000 * (39 / 39) = 350000
    expect(result).toBe(350000);
  });

  it('returns correct pro-rata salary for part-time contract', () => {
    const contract = makeContract({ weekly_hours: 20 });
    const payPlan = makePayPlan();

    const result = calculateMonthlySalary(contract, payPlan);

    // 350000 * (20 / 39) = 179487.179... => rounded to 179487
    expect(result).toBe(Math.round(350000 * (20 / 39)));
  });

  it('returns null when no active period found', () => {
    const contract = makeContract();
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2020-01-01',
          to: '2023-12-31', // ended before today (2025-06-15)
          weekly_hours: 39,
          created_at: '2020-01-01T00:00:00Z',
          updated_at: '2020-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 350000,
              created_at: '2020-01-01T00:00:00Z',
              updated_at: '2020-01-01T00:00:00Z',
            },
          ],
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('returns null when no matching entry for grade/step', () => {
    const contract = makeContract({ grade: 'S9', step: 5 });
    const payPlan = makePayPlan(); // only has S8a/step 3

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('returns null when period weekly_hours is zero', () => {
    const contract = makeContract();
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2024-01-01',
          to: null,
          weekly_hours: 0,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 350000,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('handles period with end date that is still active', () => {
    const contract = makeContract({ weekly_hours: 39 });
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2024-01-01',
          to: '2026-12-31', // still active (today is 2025-06-15)
          weekly_hours: 39,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 360000,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBe(360000);
  });

  it('picks the active period from multiple periods', () => {
    const contract = makeContract({ weekly_hours: 39 });
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2020-01-01',
          to: '2023-12-31',
          weekly_hours: 39,
          created_at: '2020-01-01T00:00:00Z',
          updated_at: '2020-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 300000, // old amount
              created_at: '2020-01-01T00:00:00Z',
              updated_at: '2020-01-01T00:00:00Z',
            },
          ],
        },
        {
          id: 2,
          payplan_id: 1,
          from: '2024-01-01',
          to: null,
          weekly_hours: 39,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 2,
              period_id: 2,
              grade: 'S8a',
              step: 3,
              monthly_amount: 350000, // current amount
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
        },
      ],
    });

    // Should use the second period (active one), not the expired first
    expect(calculateMonthlySalary(contract, payPlan)).toBe(350000);
  });

  it('returns null when payplan has no periods', () => {
    const contract = makeContract();
    const payPlan = makePayPlan({ periods: [] });

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('returns null when payplan periods is undefined', () => {
    const contract = makeContract();
    const payPlan = makePayPlan({ periods: undefined });

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('returns null when period entries is empty', () => {
    const contract = makeContract();
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2024-01-01',
          to: null,
          weekly_hours: 39,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: [],
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('returns null when period entries is undefined', () => {
    const contract = makeContract();
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2024-01-01',
          to: null,
          weekly_hours: 39,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: undefined,
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('matches grade case-sensitively', () => {
    const contract = makeContract({ grade: 's8a' }); // lowercase
    const payPlan = makePayPlan(); // has 'S8a' (uppercase)

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('rounds the result to the nearest integer', () => {
    const contract = makeContract({ weekly_hours: 30 });
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2024-01-01',
          to: null,
          weekly_hours: 39,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 350047, // produces a fractional result
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
        },
      ],
    });

    const result = calculateMonthlySalary(contract, payPlan);
    // 350047 * (30 / 39) = 269267.076... => 269267
    expect(result).toBe(Math.round(350047 * (30 / 39)));
    expect(Number.isInteger(result)).toBe(true);
  });

  it('returns 0 when contract weekly_hours is 0', () => {
    const contract = makeContract({ weekly_hours: 0 });
    const payPlan = makePayPlan();

    // 350000 * (0 / 39) = 0
    expect(calculateMonthlySalary(contract, payPlan)).toBe(0);
  });

  it('handles period starting exactly today', () => {
    const contract = makeContract({ weekly_hours: 39 });
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2025-06-15', // exactly today
          to: null,
          weekly_hours: 39,
          created_at: '2025-06-15T00:00:00Z',
          updated_at: '2025-06-15T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 370000,
              created_at: '2025-06-15T00:00:00Z',
              updated_at: '2025-06-15T00:00:00Z',
            },
          ],
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBe(370000);
  });

  it('handles period ending exactly today', () => {
    const contract = makeContract({ weekly_hours: 39 });
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2024-01-01',
          to: '2025-06-15', // ends exactly today - should still be active
          weekly_hours: 39,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 340000,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBe(340000);
  });

  it('handles period starting tomorrow (future period)', () => {
    const contract = makeContract({ weekly_hours: 39 });
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2025-06-16', // tomorrow
          to: null,
          weekly_hours: 39,
          created_at: '2025-06-16T00:00:00Z',
          updated_at: '2025-06-16T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 370000,
              created_at: '2025-06-16T00:00:00Z',
              updated_at: '2025-06-16T00:00:00Z',
            },
          ],
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBeNull();
  });

  it('correctly selects matching entry among multiple entries', () => {
    const contract = makeContract({ grade: 'S8b', step: 4, weekly_hours: 39 });
    const payPlan = makePayPlan({
      periods: [
        {
          id: 1,
          payplan_id: 1,
          from: '2024-01-01',
          to: null,
          weekly_hours: 39,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              grade: 'S8a',
              step: 3,
              monthly_amount: 350000,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
            {
              id: 2,
              period_id: 1,
              grade: 'S8b',
              step: 4,
              monthly_amount: 400000,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
            {
              id: 3,
              period_id: 1,
              grade: 'S8b',
              step: 3,
              monthly_amount: 380000,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
        },
      ],
    });

    expect(calculateMonthlySalary(contract, payPlan)).toBe(400000);
  });
});
