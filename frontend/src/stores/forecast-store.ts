import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { ForecastRequest, ForecastChild, ForecastEmployee } from '@/lib/api/types';
import { formatDateForApi } from '@/lib/utils/formatting';

/**
 * Thrown by `buildRequest()` when a required date can't be converted to
 * the API's RFC3339 wire format. The previous implementation used a
 * non-null assertion (`formatDateForApi(d)!`) and silently sent the
 * literal string "null" — backend rejected with an opaque error and the
 * user had no idea which field was at fault. ForecastBuildError carries
 * the field path (e.g. `add_children[0].contracts[1].from`) so the page
 * can show "Field X is missing or invalid".
 */
export class ForecastBuildError extends Error {
  field: string;
  constructor(field: string, message: string) {
    super(message);
    this.field = field;
    this.name = 'ForecastBuildError';
  }
}

interface ForecastState {
  // The org this scenario belongs to. Persisted alongside the rest so
  // we can detect "the user navigated to a different org" on rehydrate
  // and wipe the scenario instead of bleeding it across orgs.
  orgId: number | null;

  // Filters
  from: string | null;
  to: string | null;
  sectionId: number | undefined;

  // Overlay arrays (mirror ForecastRequest)
  addChildren: ForecastChild[];
  removeChildIds: number[];
  addEmployees: ForecastEmployee[];
  removeEmployeeIds: number[];

  // Actions - org context
  setOrgId: (orgId: number) => void;

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
  orgId: null as number | null,
  from: null as string | null,
  to: null as string | null,
  sectionId: undefined as number | undefined,
  addChildren: [] as ForecastChild[],
  removeChildIds: [] as number[],
  addEmployees: [] as ForecastEmployee[],
  removeEmployeeIds: [] as number[],
};

export const useForecastStore = create<ForecastState>()(
  persist(
    (set, get) => ({
      ...initialState,

      // Cross-org isolation: a single localStorage scenario MUST NOT
      // bleed between orgs. When the page mounts under a different
      // orgId than the one we last persisted under, drop everything
      // and start fresh. (First-time seed → just record the orgId.)
      setOrgId: (orgId) =>
        set((s) => {
          if (s.orgId !== null && s.orgId !== orgId) {
            return { ...initialState, orgId };
          }
          return { orgId };
        }),

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

        // apiDate converts a UI YYYY-MM-DD string to the backend's RFC3339
        // wire format. The previous `formatDateForApi(date)!` non-null
        // assertion silently produced JSON `null` for missing/invalid input
        // — backend rejected with an opaque "BadRequest" and the user was
        // stuck. Now: throw ForecastBuildError with the offending field
        // path; the page (handleCalculate) catches and surfaces a clear
        // field-level error.
        const apiDate = (date: string, field: string): string => {
          const r = formatDateForApi(date);
          if (r === null) {
            throw new ForecastBuildError(field, `${field}: missing or invalid date "${date}"`);
          }
          return r;
        };
        const apiDateOptional = (date?: string | null): string | undefined =>
          date ? (formatDateForApi(date) ?? undefined) : undefined;

        if (s.from) req.from = apiDate(s.from, 'from');
        if (s.to) req.to = apiDate(s.to, 'to');
        if (s.sectionId) req.section_id = s.sectionId;

        if (s.addChildren.length > 0)
          req.add_children = s.addChildren.map((c, i) => ({
            ...c,
            birthdate: apiDate(c.birthdate, `add_children[${i}].birthdate`),
            contracts: c.contracts.map((ct, j) => ({
              ...ct,
              from: apiDate(ct.from, `add_children[${i}].contracts[${j}].from`),
              to: apiDateOptional(ct.to),
            })),
          }));
        if (s.removeChildIds.length > 0) req.remove_child_ids = s.removeChildIds;
        if (s.addEmployees.length > 0)
          req.add_employees = s.addEmployees.map((e, i) => ({
            ...e,
            birthdate: apiDate(e.birthdate, `add_employees[${i}].birthdate`),
            contracts: e.contracts.map((ct, j) => ({
              ...ct,
              from: apiDate(ct.from, `add_employees[${i}].contracts[${j}].from`),
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
    }),
    {
      // Single localStorage key, scenario isolated by orgId via the
      // setOrgId action above. Bumping the version invalidates any
      // saved scenarios from older shapes — bump deliberately when
      // the persisted shape changes.
      name: 'kitamanager-forecast-scenario',
      version: 1,
    }
  )
);
