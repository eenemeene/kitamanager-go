import { QueryClient } from '@tanstack/react-query';
import { queryKeys } from '../queryKeys';

/**
 * Tests that invalidating governmentFundings.all() invalidates all
 * sub-queries including lookup/lookupDetail used by the funding
 * attributes hook. This validates the fix where period/property mutations
 * only invalidated the detail key, leaving the lookup cache stale.
 */
describe('governmentFundings cache invalidation', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('invalidating governmentFundings.all() marks all sub-queries as stale', () => {
    // Seed caches used by both the rates page and the funding attributes hook
    queryClient.setQueryData(queryKeys.governmentFundings.list(1), {
      data: [{ id: 1 }],
      total: 1,
    });
    queryClient.setQueryData(queryKeys.governmentFundings.detail(1), {
      id: 1,
      periods: [],
    });
    queryClient.setQueryData(queryKeys.governmentFundings.lookup(), {
      data: [{ id: 1, state: 'BE' }],
    });
    queryClient.setQueryData(queryKeys.governmentFundings.lookupDetail(1), {
      id: 1,
      periods: [{ id: 10, properties: [] }],
    });

    // Invalidate using the prefix key (what period/property mutations now do)
    queryClient.invalidateQueries({ queryKey: queryKeys.governmentFundings.all() });

    expect(queryClient.getQueryState(queryKeys.governmentFundings.list(1))?.isInvalidated).toBe(
      true
    );
    expect(queryClient.getQueryState(queryKeys.governmentFundings.detail(1))?.isInvalidated).toBe(
      true
    );
    expect(queryClient.getQueryState(queryKeys.governmentFundings.lookup())?.isInvalidated).toBe(
      true
    );
    expect(
      queryClient.getQueryState(queryKeys.governmentFundings.lookupDetail(1))?.isInvalidated
    ).toBe(true);
  });

  it('invalidating detail(id) does NOT invalidate lookupDetail(id)', () => {
    queryClient.setQueryData(queryKeys.governmentFundings.detail(1), { id: 1 });
    queryClient.setQueryData(queryKeys.governmentFundings.lookupDetail(1), { id: 1 });

    // Old behavior: only invalidate detail
    queryClient.invalidateQueries({ queryKey: queryKeys.governmentFundings.detail(1) });

    // detail is invalidated
    expect(queryClient.getQueryState(queryKeys.governmentFundings.detail(1))?.isInvalidated).toBe(
      true
    );
    // lookupDetail is NOT — this is the bug the fix addresses
    expect(
      queryClient.getQueryState(queryKeys.governmentFundings.lookupDetail(1))?.isInvalidated
    ).toBe(false);
  });
});
