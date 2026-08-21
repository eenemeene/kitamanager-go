'use client';

import { useState, useCallback } from 'react';
import { useDebouncedValue } from './use-debounced-value';
import { todayBerlinString } from '@/lib/utils/contracts';

interface UseResourceListFiltersConfig {
  /** Debounce delay for search input in ms (default: 300) */
  debounceMs?: number;
}

/**
 * Shared hook for resource list page filter state:
 * pagination, debounced search, date filter, and a generic category filter.
 *
 * All setters automatically reset the page to 1.
 */
export function useResourceListFilters({ debounceMs = 300 }: UseResourceListFiltersConfig = {}) {
  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState('');
  const search = useDebouncedValue(searchInput, debounceMs);
  // Local midnight on *Berlin's* today, not the browser's. The value is read
  // back with `toLocalDateString`, so anchoring it this way makes the filter
  // agree with `models.Today()` on the server -- which is what decides whether
  // a contract counts as active. A user in a zone behind Berlin would otherwise
  // open the page already filtered to yesterday.
  const [activeOn, setActiveOn] = useState(() => new Date(`${todayBerlinString()}T00:00:00`));

  const setSearchAndResetPage = useCallback((value: string) => {
    setSearchInput(value);
    setPage(1);
  }, []);

  const setActiveOnAndResetPage = useCallback((date: Date) => {
    setActiveOn(date);
    setPage(1);
  }, []);

  return {
    page,
    setPage,
    searchInput,
    setSearchInput: setSearchAndResetPage,
    search,
    activeOn,
    setActiveOn: setActiveOnAndResetPage,
  };
}
