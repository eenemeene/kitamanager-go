import { QueryClient } from '@tanstack/react-query';
import { queryKeys } from '../queryKeys';

/**
 * Tests that invalidating children.all(orgId) invalidates all children
 * sub-queries. This validates the fix where the dashboard acceptSuggestion
 * mutation previously only invalidated withoutVouchers, leaving the child
 * name, billingSummary, funding, and list caches stale.
 */
describe('children cache invalidation', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('invalidating children.all(orgId) marks all children sub-queries as stale', () => {
    const orgId = 1;

    // Seed the cache with various children-related data
    queryClient.setQueryData(queryKeys.children.withoutVouchers(orgId), [{ id: 1 }]);
    queryClient.setQueryData(queryKeys.children.billingSummary(orgId), { total: 100 });
    queryClient.setQueryData(queryKeys.children.funding(orgId), { children: [] });
    queryClient.setQueryData(queryKeys.children.detail(orgId, 5), { id: 5, first_name: 'Old' });
    queryClient.setQueryData(queryKeys.children.allUnpaginated(orgId), [{ id: 5 }]);

    // Invalidate using the prefix key
    queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });

    expect(
      queryClient.getQueryState(queryKeys.children.withoutVouchers(orgId))?.isInvalidated
    ).toBe(true);
    expect(queryClient.getQueryState(queryKeys.children.billingSummary(orgId))?.isInvalidated).toBe(
      true
    );
    expect(queryClient.getQueryState(queryKeys.children.funding(orgId))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(queryKeys.children.detail(orgId, 5))?.isInvalidated).toBe(
      true
    );
    expect(queryClient.getQueryState(queryKeys.children.allUnpaginated(orgId))?.isInvalidated).toBe(
      true
    );
  });

  it('invalidating children.all(orgId) does NOT invalidate another org', () => {
    queryClient.setQueryData(queryKeys.children.withoutVouchers(1), [{ id: 1 }]);
    queryClient.setQueryData(queryKeys.children.withoutVouchers(2), [{ id: 2 }]);

    queryClient.invalidateQueries({ queryKey: queryKeys.children.all(1) });

    expect(queryClient.getQueryState(queryKeys.children.withoutVouchers(1))?.isInvalidated).toBe(
      true
    );
    expect(queryClient.getQueryState(queryKeys.children.withoutVouchers(2))?.isInvalidated).toBe(
      false
    );
  });
});
