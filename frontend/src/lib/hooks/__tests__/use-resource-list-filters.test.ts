import { renderHook, act } from '@testing-library/react';
import { useResourceListFilters } from '../use-resource-list-filters';

describe('useResourceListFilters', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('initializes with page 1 and empty search', () => {
    const { result } = renderHook(() => useResourceListFilters());
    expect(result.current.page).toBe(1);
    expect(result.current.searchInput).toBe('');
    expect(result.current.search).toBe('');
  });

  it('sets page', () => {
    const { result } = renderHook(() => useResourceListFilters());
    act(() => result.current.setPage(3));
    expect(result.current.page).toBe(3);
  });

  it('resets page to 1 when search changes', () => {
    const { result } = renderHook(() => useResourceListFilters());

    act(() => result.current.setPage(3));
    expect(result.current.page).toBe(3);

    act(() => result.current.setSearchInput('test'));
    expect(result.current.page).toBe(1);
    expect(result.current.searchInput).toBe('test');
  });

  it('debounces search value', () => {
    const { result } = renderHook(() => useResourceListFilters({ debounceMs: 300 }));

    act(() => result.current.setSearchInput('hello'));
    expect(result.current.searchInput).toBe('hello');
    expect(result.current.search).toBe(''); // not yet debounced

    act(() => jest.advanceTimersByTime(300));
    expect(result.current.search).toBe('hello');
  });

  it('resets page to 1 when activeOn changes', () => {
    const { result } = renderHook(() => useResourceListFilters());

    act(() => result.current.setPage(5));
    expect(result.current.page).toBe(5);

    act(() => result.current.setActiveOn(new Date('2025-06-01')));
    expect(result.current.page).toBe(1);
  });

  it('initializes activeOn to Berlin today, not the browser day', () => {
    // The default feeds `active_on`, which the server evaluates against
    // `models.Today()`. Pinned to 00:30 on 1 August in Berlin -- still 31 July
    // in UTC -- so the two answers differ and a browser-clock default fails.
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-07-31T22:30:00Z'));
    try {
      const { result } = renderHook(() => useResourceListFilters());

      const activeOn = result.current.activeOn;
      expect(activeOn.getFullYear()).toBe(2026);
      expect(activeOn.getMonth()).toBe(7);
      expect(activeOn.getDate()).toBe(1);
    } finally {
      jest.useRealTimers();
    }
  });
});
