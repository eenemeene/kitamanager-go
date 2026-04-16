import { QueryClient } from '@tanstack/react-query';
import { queryKeys } from '../queryKeys';

/**
 * Tests that invalidating attendance.all(orgId) invalidates all attendance
 * sub-queries (byDate, summary, byWeek). This validates the fix where
 * attendance mutations previously only invalidated the specific date,
 * leaving summary and week caches stale.
 */
describe('attendance cache invalidation', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('invalidating attendance.all(orgId) marks all attendance sub-queries as stale', () => {
    const orgId = 1;

    // Seed the cache with attendance data at various keys
    queryClient.setQueryData(queryKeys.attendance.byDate(orgId, '2025-06-15'), [
      { id: 1, status: 'present' },
    ]);
    queryClient.setQueryData(queryKeys.attendance.byDate(orgId, '2025-06-16'), [
      { id: 2, status: 'absent' },
    ]);
    queryClient.setQueryData(queryKeys.attendance.summary(orgId, '2025-06-15'), {
      present: 10,
      absent: 2,
    });
    queryClient.setQueryData(queryKeys.attendance.byWeek(orgId, '2025-06-09'), [
      { id: 3, status: 'present' },
    ]);

    // Invalidate using the prefix key (what attendance mutations now do)
    queryClient.invalidateQueries({ queryKey: queryKeys.attendance.all(orgId) });

    // All attendance queries for this org should be invalidated (stale)
    const byDate15 = queryClient.getQueryState(queryKeys.attendance.byDate(orgId, '2025-06-15'));
    const byDate16 = queryClient.getQueryState(queryKeys.attendance.byDate(orgId, '2025-06-16'));
    const summary = queryClient.getQueryState(queryKeys.attendance.summary(orgId, '2025-06-15'));
    const byWeek = queryClient.getQueryState(queryKeys.attendance.byWeek(orgId, '2025-06-09'));

    expect(byDate15?.isInvalidated).toBe(true);
    expect(byDate16?.isInvalidated).toBe(true);
    expect(summary?.isInvalidated).toBe(true);
    expect(byWeek?.isInvalidated).toBe(true);
  });

  it('invalidating attendance.all(orgId) does NOT invalidate another org', () => {
    // Seed data for two different orgs
    queryClient.setQueryData(queryKeys.attendance.byDate(1, '2025-06-15'), [{ id: 1 }]);
    queryClient.setQueryData(queryKeys.attendance.byDate(2, '2025-06-15'), [{ id: 2 }]);

    // Invalidate only org 1
    queryClient.invalidateQueries({ queryKey: queryKeys.attendance.all(1) });

    const org1 = queryClient.getQueryState(queryKeys.attendance.byDate(1, '2025-06-15'));
    const org2 = queryClient.getQueryState(queryKeys.attendance.byDate(2, '2025-06-15'));

    expect(org1?.isInvalidated).toBe(true);
    expect(org2?.isInvalidated).toBe(false);
  });
});
