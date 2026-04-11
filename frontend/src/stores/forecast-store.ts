import { create } from 'zustand';
import type {
  ForecastRequest,
  ForecastAddChild,
  ForecastAddEmployee,
  ForecastAddPayPlanPeriod,
  ForecastAddFundingPeriod,
  ForecastAddBudgetItem,
  PayPlan,
} from '@/lib/api/types';
import { formatDateForApi } from '@/lib/utils/formatting';

interface ForecastState {
  // Filters
  from: string | null;
  to: string | null;
  sectionId: number | undefined;

  // Overlay arrays (mirror ForecastRequest)
  addChildren: ForecastAddChild[];
  removeChildIds: number[];
  addEmployees: ForecastAddEmployee[];
  removeEmployeeIds: number[];
  addPayPlanPeriods: ForecastAddPayPlanPeriod[];
  addFundingPeriods: ForecastAddFundingPeriod[];
  addBudgetItems: ForecastAddBudgetItem[];
  removeBudgetItemIds: number[];

  // UI-only state
  salaryIncreasePercent: number | null;
  salaryEffectiveFrom: string | null;

  // Actions - filters
  setFilters: (from: string | null, to: string | null, sectionId?: number) => void;

  // Actions - children
  addChild: (child: ForecastAddChild) => void;
  removeAddedChild: (index: number) => void;
  toggleRemoveChild: (childId: number) => void;

  // Actions - employees
  addEmployee: (employee: ForecastAddEmployee) => void;
  removeAddedEmployee: (index: number) => void;
  toggleRemoveEmployee: (employeeId: number) => void;

  // Actions - salary
  setSalaryIncrease: (
    percent: number | null,
    effectiveFrom: string | null,
    payPlans: PayPlan[]
  ) => void;

  // Actions - funding
  addFundingPeriod: (period: ForecastAddFundingPeriod) => void;
  removeFundingPeriod: (index: number) => void;

  // Actions - budget
  addBudgetItem: (item: ForecastAddBudgetItem) => void;
  removeAddedBudgetItem: (index: number) => void;
  toggleRemoveBudgetItem: (budgetItemId: number) => void;

  // Helpers
  buildRequest: () => ForecastRequest;
  reset: () => void;
  hasModifications: () => boolean;
  modificationCount: () => number;
}

const initialState = {
  from: null as string | null,
  to: null as string | null,
  sectionId: undefined as number | undefined,
  addChildren: [] as ForecastAddChild[],
  removeChildIds: [] as number[],
  addEmployees: [] as ForecastAddEmployee[],
  removeEmployeeIds: [] as number[],
  addPayPlanPeriods: [] as ForecastAddPayPlanPeriod[],
  addFundingPeriods: [] as ForecastAddFundingPeriod[],
  addBudgetItems: [] as ForecastAddBudgetItem[],
  removeBudgetItemIds: [] as number[],
  salaryIncreasePercent: null as number | null,
  salaryEffectiveFrom: null as string | null,
};

export const useForecastStore = create<ForecastState>()((set, get) => ({
  ...initialState,

  setFilters: (from, to, sectionId) => set({ from, to, sectionId }),

  // Children actions
  addChild: (child) => set((s) => ({ addChildren: [...s.addChildren, child] })),
  removeAddedChild: (index) =>
    set((s) => ({ addChildren: s.addChildren.filter((_, i) => i !== index) })),
  toggleRemoveChild: (childId) =>
    set((s) => ({
      removeChildIds: s.removeChildIds.includes(childId)
        ? s.removeChildIds.filter((id) => id !== childId)
        : [...s.removeChildIds, childId],
    })),

  // Employee actions
  addEmployee: (employee) => set((s) => ({ addEmployees: [...s.addEmployees, employee] })),
  removeAddedEmployee: (index) =>
    set((s) => ({ addEmployees: s.addEmployees.filter((_, i) => i !== index) })),
  toggleRemoveEmployee: (employeeId) =>
    set((s) => ({
      removeEmployeeIds: s.removeEmployeeIds.includes(employeeId)
        ? s.removeEmployeeIds.filter((id) => id !== employeeId)
        : [...s.removeEmployeeIds, employeeId],
    })),

  // Salary increase: generates addPayPlanPeriods from current pay plan data
  setSalaryIncrease: (percent, effectiveFrom, payPlans) => {
    if (percent === null || !effectiveFrom || payPlans.length === 0) {
      set({
        salaryIncreasePercent: percent,
        salaryEffectiveFrom: effectiveFrom,
        addPayPlanPeriods: [],
      });
      return;
    }

    const multiplier = 1 + percent / 100;
    const periods: ForecastAddPayPlanPeriod[] = [];

    for (const pp of payPlans) {
      if (!pp.periods?.length) continue;

      // Find the latest period (most recent "from" date) as the basis
      const sorted = [...pp.periods].sort(
        (a, b) => new Date(b.from).getTime() - new Date(a.from).getTime()
      );
      const latestPeriod = sorted[0];
      if (!latestPeriod.entries?.length) continue;

      periods.push({
        pay_plan_id: pp.id,
        from: effectiveFrom,
        weekly_hours: latestPeriod.weekly_hours,
        employer_contribution_rate: latestPeriod.employer_contribution_rate,
        entries: latestPeriod.entries.map((e) => ({
          grade: e.grade,
          step: e.step,
          monthly_amount: Math.round(e.monthly_amount * multiplier),
        })),
      });
    }

    set({
      salaryIncreasePercent: percent,
      salaryEffectiveFrom: effectiveFrom,
      addPayPlanPeriods: periods,
    });
  },

  // Funding actions
  addFundingPeriod: (period) =>
    set((s) => ({ addFundingPeriods: [...s.addFundingPeriods, period] })),
  removeFundingPeriod: (index) =>
    set((s) => ({ addFundingPeriods: s.addFundingPeriods.filter((_, i) => i !== index) })),

  // Budget actions
  addBudgetItem: (item) => set((s) => ({ addBudgetItems: [...s.addBudgetItems, item] })),
  removeAddedBudgetItem: (index) =>
    set((s) => ({ addBudgetItems: s.addBudgetItems.filter((_, i) => i !== index) })),
  toggleRemoveBudgetItem: (budgetItemId) =>
    set((s) => ({
      removeBudgetItemIds: s.removeBudgetItemIds.includes(budgetItemId)
        ? s.removeBudgetItemIds.filter((id) => id !== budgetItemId)
        : [...s.removeBudgetItemIds, budgetItemId],
    })),

  buildRequest: (): ForecastRequest => {
    const s = get();
    const req: ForecastRequest = {};

    const apiDate = (date: string): string => formatDateForApi(date)!;
    const apiDateOptional = (date?: string | null): string | undefined =>
      date ? (formatDateForApi(date) ?? undefined) : undefined;

    if (s.from) req.from = apiDate(s.from);
    if (s.to) req.to = apiDate(s.to);
    if (s.sectionId) req.section_id = s.sectionId;

    if (s.addChildren.length > 0)
      req.add_children = s.addChildren.map((c) => ({
        ...c,
        birthdate: apiDate(c.birthdate),
        contracts: c.contracts.map((ct) => ({
          ...ct,
          from: apiDate(ct.from),
          to: apiDateOptional(ct.to),
        })),
      }));
    if (s.removeChildIds.length > 0) req.remove_child_ids = s.removeChildIds;
    if (s.addEmployees.length > 0)
      req.add_employees = s.addEmployees.map((e) => ({
        ...e,
        birthdate: apiDate(e.birthdate),
        contracts: e.contracts.map((ct) => ({
          ...ct,
          from: apiDate(ct.from),
          to: apiDateOptional(ct.to),
        })),
      }));
    if (s.removeEmployeeIds.length > 0) req.remove_employee_ids = s.removeEmployeeIds;
    if (s.addPayPlanPeriods.length > 0)
      req.add_pay_plan_periods = s.addPayPlanPeriods.map((p) => ({
        ...p,
        from: apiDate(p.from),
        to: apiDateOptional(p.to),
      }));
    if (s.addFundingPeriods.length > 0)
      req.add_funding_periods = s.addFundingPeriods.map((f) => ({
        ...f,
        from: apiDate(f.from),
        to: apiDateOptional(f.to),
      }));
    if (s.addBudgetItems.length > 0)
      req.add_budget_items = s.addBudgetItems.map((b) => ({
        ...b,
        entries: b.entries.map((e) => ({
          ...e,
          from: apiDate(e.from),
          to: apiDateOptional(e.to),
        })),
      }));
    if (s.removeBudgetItemIds.length > 0) req.remove_budget_item_ids = s.removeBudgetItemIds;

    return req;
  },

  reset: () => set(initialState),

  hasModifications: () => {
    const s = get();
    return (
      s.addChildren.length > 0 ||
      s.removeChildIds.length > 0 ||
      s.addEmployees.length > 0 ||
      s.removeEmployeeIds.length > 0 ||
      s.addPayPlanPeriods.length > 0 ||
      s.addFundingPeriods.length > 0 ||
      s.addBudgetItems.length > 0 ||
      s.removeBudgetItemIds.length > 0
    );
  },

  modificationCount: () => {
    const s = get();
    return (
      s.addChildren.length +
      s.removeChildIds.length +
      s.addEmployees.length +
      s.removeEmployeeIds.length +
      s.addPayPlanPeriods.length +
      s.addFundingPeriods.length +
      s.addBudgetItems.length +
      s.removeBudgetItemIds.length
    );
  },
}));
