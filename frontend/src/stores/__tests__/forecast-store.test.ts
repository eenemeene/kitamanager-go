import { useForecastStore } from '../forecast-store';
import type { ForecastAddChild, ForecastAddEmployee, PayPlan } from '@/lib/api/types';

describe('forecast-store', () => {
  beforeEach(() => {
    useForecastStore.getState().reset();
  });

  it('starts with empty state and no modifications', () => {
    const state = useForecastStore.getState();
    expect(state.hasModifications()).toBe(false);
    expect(state.modificationCount()).toBe(0);
    expect(state.addChildren).toEqual([]);
    expect(state.removeChildIds).toEqual([]);
  });

  it('sets filters', () => {
    useForecastStore.getState().setFilters('2026-01-01', '2026-12-01', 5);
    const state = useForecastStore.getState();
    expect(state.from).toBe('2026-01-01');
    expect(state.to).toBe('2026-12-01');
    expect(state.sectionId).toBe(5);
  });

  // Children actions
  it('adds and removes children', () => {
    const child: ForecastAddChild = {
      first_name: 'Child',
      last_name: '#1',
      gender: 'diverse',
      birthdate: '2023-01-01',
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    };
    useForecastStore.getState().addChild(child);
    expect(useForecastStore.getState().addChildren).toHaveLength(1);
    expect(useForecastStore.getState().hasModifications()).toBe(true);
    expect(useForecastStore.getState().modificationCount()).toBe(1);

    useForecastStore.getState().removeAddedChild(0);
    expect(useForecastStore.getState().addChildren).toHaveLength(0);
  });

  it('toggles remove child', () => {
    useForecastStore.getState().toggleRemoveChild(42);
    expect(useForecastStore.getState().removeChildIds).toEqual([42]);

    useForecastStore.getState().toggleRemoveChild(42);
    expect(useForecastStore.getState().removeChildIds).toEqual([]);
  });

  // Employee actions
  it('adds and removes employees', () => {
    const employee: ForecastAddEmployee = {
      first_name: 'Employee',
      last_name: '#1',
      gender: 'female',
      birthdate: '1990-01-01',
      contracts: [
        {
          from: '2026-08-01',
          section_id: 1,
          staff_category: 'qualified',
          weekly_hours: 39,
          pay_plan_id: 1,
        },
      ],
    };
    useForecastStore.getState().addEmployee(employee);
    expect(useForecastStore.getState().addEmployees).toHaveLength(1);
    expect(useForecastStore.getState().modificationCount()).toBe(1);

    useForecastStore.getState().removeAddedEmployee(0);
    expect(useForecastStore.getState().addEmployees).toHaveLength(0);
  });

  it('toggles remove employee', () => {
    useForecastStore.getState().toggleRemoveEmployee(10);
    expect(useForecastStore.getState().removeEmployeeIds).toEqual([10]);

    useForecastStore.getState().toggleRemoveEmployee(10);
    expect(useForecastStore.getState().removeEmployeeIds).toEqual([]);
  });

  // Salary increase
  it('generates pay plan periods from salary increase', () => {
    const payPlans: PayPlan[] = [
      {
        id: 1,
        organization_id: 1,
        name: 'TVöD SuE',
        created_at: '',
        updated_at: '',
        periods: [
          {
            id: 1,
            payplan_id: 1,
            from: '2024-01-01',
            weekly_hours: 39,
            employer_contribution_rate: 2200,
            created_at: '',
            updated_at: '',
            entries: [
              {
                id: 1,
                period_id: 1,
                grade: 'S8a',
                step: 3,
                monthly_amount: 380000,
                created_at: '',
                updated_at: '',
              },
            ],
          },
        ],
      },
    ];

    useForecastStore.getState().setSalaryIncrease(10, '2027-01-01', payPlans);
    const state = useForecastStore.getState();
    expect(state.salaryIncreasePercent).toBe(10);
    expect(state.salaryEffectiveFrom).toBe('2027-01-01');
    expect(state.addPayPlanPeriods).toHaveLength(1);
    expect(state.addPayPlanPeriods[0].pay_plan_id).toBe(1);
    expect(state.addPayPlanPeriods[0].from).toBe('2027-01-01');
    expect(state.addPayPlanPeriods[0].entries[0].monthly_amount).toBe(418000); // 380000 * 1.1
  });

  it('clears pay plan periods when percent is null', () => {
    const payPlans: PayPlan[] = [];
    useForecastStore.getState().setSalaryIncrease(null, null, payPlans);
    expect(useForecastStore.getState().addPayPlanPeriods).toEqual([]);
  });

  // Funding actions
  it('adds and removes funding periods', () => {
    useForecastStore.getState().addFundingPeriod({
      from: '2027-08-01',
      full_time_weekly_hours: 39,
      properties: [],
    });
    expect(useForecastStore.getState().addFundingPeriods).toHaveLength(1);

    useForecastStore.getState().removeFundingPeriod(0);
    expect(useForecastStore.getState().addFundingPeriods).toHaveLength(0);
  });

  // Budget actions
  it('adds and removes budget items', () => {
    useForecastStore.getState().addBudgetItem({
      name: 'Test',
      category: 'income',
      per_child: false,
      entries: [{ from: '2027-01-01', amount_cents: 50000 }],
    });
    expect(useForecastStore.getState().addBudgetItems).toHaveLength(1);

    useForecastStore.getState().removeAddedBudgetItem(0);
    expect(useForecastStore.getState().addBudgetItems).toHaveLength(0);
  });

  it('toggles remove budget item', () => {
    useForecastStore.getState().toggleRemoveBudgetItem(7);
    expect(useForecastStore.getState().removeBudgetItemIds).toEqual([7]);

    useForecastStore.getState().toggleRemoveBudgetItem(7);
    expect(useForecastStore.getState().removeBudgetItemIds).toEqual([]);
  });

  // buildRequest
  it('builds a request with RFC3339 dates', () => {
    useForecastStore.getState().setFilters('2026-01-01', '2026-12-01');
    useForecastStore.getState().addChild({
      first_name: 'Child',
      last_name: '#1',
      gender: 'diverse',
      birthdate: '2023-06-15',
      contracts: [{ from: '2026-08-01', to: '2027-07-31', section_id: 1 }],
    });

    const req = useForecastStore.getState().buildRequest();
    expect(req.from).toBe('2026-01-01T00:00:00Z');
    expect(req.to).toBe('2026-12-01T00:00:00Z');
    expect(req.add_children![0].birthdate).toBe('2023-06-15T00:00:00Z');
    expect(req.add_children![0].contracts[0].from).toBe('2026-08-01T00:00:00Z');
    expect(req.add_children![0].contracts[0].to).toBe('2027-07-31T00:00:00Z');
  });

  it('builds empty request when no modifications', () => {
    const req = useForecastStore.getState().buildRequest();
    expect(req).toEqual({});
  });

  it('omits empty arrays from request', () => {
    useForecastStore.getState().setFilters('2026-01-01', '2026-12-01');
    const req = useForecastStore.getState().buildRequest();
    expect(req.add_children).toBeUndefined();
    expect(req.remove_child_ids).toBeUndefined();
    expect(req.add_employees).toBeUndefined();
  });

  // Reset
  it('resets to initial state', () => {
    useForecastStore.getState().setFilters('2026-01-01', '2026-12-01', 1);
    useForecastStore.getState().addChild({
      first_name: 'Child',
      last_name: '#1',
      gender: 'diverse',
      birthdate: '2023-01-01',
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    });
    useForecastStore.getState().toggleRemoveChild(42);

    useForecastStore.getState().reset();
    const state = useForecastStore.getState();
    expect(state.from).toBeNull();
    expect(state.to).toBeNull();
    expect(state.sectionId).toBeUndefined();
    expect(state.addChildren).toEqual([]);
    expect(state.removeChildIds).toEqual([]);
    expect(state.hasModifications()).toBe(false);
  });

  // modificationCount
  it('counts all modifications', () => {
    useForecastStore.getState().addChild({
      first_name: 'A',
      last_name: 'B',
      gender: 'diverse',
      birthdate: '2023-01-01',
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    });
    useForecastStore.getState().toggleRemoveChild(1);
    useForecastStore.getState().addEmployee({
      first_name: 'E',
      last_name: 'F',
      gender: 'female',
      birthdate: '1990-01-01',
      contracts: [
        {
          from: '2026-08-01',
          section_id: 1,
          staff_category: 'qualified',
          weekly_hours: 39,
          pay_plan_id: 1,
        },
      ],
    });
    expect(useForecastStore.getState().modificationCount()).toBe(3);
  });
});
