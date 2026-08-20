import { useForecastStore, ForecastBuildError } from '../forecast-store';
import type { ForecastChild, ForecastEmployee } from '@/lib/api/types';

describe('forecast-store', () => {
  beforeEach(() => {
    // Reset persists too — clear localStorage between tests so cross-org
    // assertions (below) don't see leftovers from a previous test.
    localStorage.clear();
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
    const child: ForecastChild = {
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
    const employee: ForecastEmployee = {
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
          payplan_id: 1,
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
    expect(req.add_children![0].contracts![0]!.from).toBe('2026-08-01T00:00:00Z');
    expect(req.add_children![0].contracts![0]!.to).toBe('2027-07-31T00:00:00Z');
  });

  it('builds empty request when no modifications', () => {
    const req = useForecastStore.getState().buildRequest();
    expect(req).toEqual({});
  });

  // F8: ForecastBuildError surfaces field paths instead of producing
  // silent JSON `null` from a non-null assertion. The previous behavior
  // sent `birthdate: null` (or even the string "null") to the backend
  // and the user saw an opaque "BadRequest".
  it('throws ForecastBuildError with field path for missing child birthdate', () => {
    useForecastStore.getState().addChild({
      first_name: 'A',
      last_name: 'B',
      gender: 'female',
      birthdate: '', // empty; previously coerced to JSON null
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    });
    let caught: unknown = null;
    try {
      useForecastStore.getState().buildRequest();
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ForecastBuildError);
    if (caught instanceof ForecastBuildError) {
      expect(caught.field).toBe('add_children[0].birthdate');
      expect(caught.message).toContain('add_children[0].birthdate');
    }
  });

  it('throws ForecastBuildError with field path for missing child contract.from', () => {
    useForecastStore.getState().addChild({
      first_name: 'A',
      last_name: 'B',
      gender: 'female',
      birthdate: '2023-06-15',
      contracts: [{ from: '', section_id: 1 }],
    });
    expect(() => useForecastStore.getState().buildRequest()).toThrow(ForecastBuildError);
    try {
      useForecastStore.getState().buildRequest();
    } catch (e) {
      if (e instanceof ForecastBuildError) {
        expect(e.field).toBe('add_children[0].contracts[0].from');
      }
    }
  });

  it('throws ForecastBuildError with field path for missing employee contract.from', () => {
    useForecastStore.getState().addEmployee({
      first_name: 'E',
      last_name: 'F',
      gender: 'female',
      birthdate: '1990-01-01',
      contracts: [
        {
          from: '',
          section_id: 1,
          staff_category: 'qualified',
          weekly_hours: 39,
          payplan_id: 1,
        },
      ],
    });
    try {
      useForecastStore.getState().buildRequest();
      throw new Error('expected throw');
    } catch (e) {
      expect(e).toBeInstanceOf(ForecastBuildError);
      if (e instanceof ForecastBuildError) {
        expect(e.field).toBe('add_employees[0].contracts[0].from');
      }
    }
  });

  it('does not throw for absent overlay (clean baseline-only request)', () => {
    useForecastStore.getState().setFilters('2026-01-01', '2026-12-01');
    expect(() => useForecastStore.getState().buildRequest()).not.toThrow();
  });

  // F9: cross-org isolation + persistence
  it('setOrgId records the org on first set without wiping', () => {
    useForecastStore.getState().addChild({
      first_name: 'A',
      last_name: 'B',
      gender: 'female',
      birthdate: '2023-01-01',
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    });
    useForecastStore.getState().setOrgId(7);
    const state = useForecastStore.getState();
    expect(state.orgId).toBe(7);
    expect(state.addChildren).toHaveLength(1);
  });

  it('setOrgId wipes the scenario when navigating to a different org', () => {
    // Seed a scenario under org 1.
    useForecastStore.getState().setOrgId(1);
    useForecastStore.getState().addChild({
      first_name: 'A',
      last_name: 'B',
      gender: 'female',
      birthdate: '2023-01-01',
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    });
    useForecastStore.getState().toggleRemoveEmployee(99);
    expect(useForecastStore.getState().modificationCount()).toBe(2);

    // Navigate to org 2 — the persisted state must NOT bleed across.
    useForecastStore.getState().setOrgId(2);
    const state = useForecastStore.getState();
    expect(state.orgId).toBe(2);
    expect(state.addChildren).toHaveLength(0);
    expect(state.removeEmployeeIds).toEqual([]);
    expect(state.modificationCount()).toBe(0);
  });

  it('setOrgId is idempotent for the same org (does not wipe on rerender)', () => {
    useForecastStore.getState().setOrgId(1);
    useForecastStore.getState().addChild({
      first_name: 'A',
      last_name: 'B',
      gender: 'female',
      birthdate: '2023-01-01',
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    });
    // Calling setOrgId(1) again (e.g. component re-renders) must
    // preserve the scenario; otherwise every rerender wipes the user's
    // work.
    useForecastStore.getState().setOrgId(1);
    expect(useForecastStore.getState().addChildren).toHaveLength(1);
  });

  it('persists state to localStorage', () => {
    useForecastStore.getState().setOrgId(1);
    useForecastStore.getState().addChild({
      first_name: 'Persist',
      last_name: 'Me',
      gender: 'female',
      birthdate: '2023-01-01',
      contracts: [{ from: '2026-08-01', section_id: 1 }],
    });
    // Persist middleware writes synchronously after each set in tests
    // (the default storage is localStorage, which is synchronous).
    const raw = localStorage.getItem('kitamanager-forecast-scenario');
    expect(raw).not.toBeNull();
    const parsed = JSON.parse(raw!);
    expect(parsed.state.orgId).toBe(1);
    expect(parsed.state.addChildren).toHaveLength(1);
    expect(parsed.state.addChildren[0].first_name).toBe('Persist');
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
          payplan_id: 1,
        },
      ],
    });
    expect(useForecastStore.getState().modificationCount()).toBe(3);
  });
});
