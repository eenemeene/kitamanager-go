import { create } from 'zustand';
import type { ForecastRequest, ForecastChild, ForecastEmployee } from '@/lib/api/types';
import { formatDateForApi } from '@/lib/utils/formatting';

interface ForecastState {
  // Filters
  from: string | null;
  to: string | null;
  sectionId: number | undefined;

  // Overlay arrays (mirror ForecastRequest)
  addChildren: ForecastChild[];
  removeChildIds: number[];
  addEmployees: ForecastEmployee[];
  removeEmployeeIds: number[];

  // Actions - filters
  setFilters: (from: string | null, to: string | null, sectionId?: number) => void;

  // Actions - children
  addChild: (child: ForecastChild) => void;
  removeAddedChild: (index: number) => void;
  toggleRemoveChild: (childId: number) => void;

  // Actions - employees
  addEmployee: (employee: ForecastEmployee) => void;
  removeAddedEmployee: (index: number) => void;
  toggleRemoveEmployee: (employeeId: number) => void;

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
  addChildren: [] as ForecastChild[],
  removeChildIds: [] as number[],
  addEmployees: [] as ForecastEmployee[],
  removeEmployeeIds: [] as number[],
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

    return req;
  },

  reset: () => set(initialState),

  hasModifications: () => {
    const s = get();
    return (
      s.addChildren.length > 0 ||
      s.removeChildIds.length > 0 ||
      s.addEmployees.length > 0 ||
      s.removeEmployeeIds.length > 0
    );
  },

  modificationCount: () => {
    const s = get();
    return (
      s.addChildren.length +
      s.removeChildIds.length +
      s.addEmployees.length +
      s.removeEmployeeIds.length
    );
  },
}));
