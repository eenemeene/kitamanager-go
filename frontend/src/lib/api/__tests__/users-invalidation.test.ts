import { QueryClient } from '@tanstack/react-query';
import { queryKeys } from '../queryKeys';

/**
 * Tests that invalidating users.all() invalidates all user sub-queries.
 * This validates the fix where the superadmin mutation used a hardcoded
 * ['users'] key instead of the queryKeys factory.
 */
describe('users cache invalidation', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('invalidating users.all() marks list and memberships as stale', () => {
    queryClient.setQueryData(queryKeys.users.list(1), { data: [{ id: 1 }], total: 1 });
    queryClient.setQueryData(queryKeys.users.memberships(1), [{ org_id: 1, role: 'admin' }]);

    queryClient.invalidateQueries({ queryKey: queryKeys.users.all() });

    expect(queryClient.getQueryState(queryKeys.users.list(1))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(queryKeys.users.memberships(1))?.isInvalidated).toBe(true);
  });
});
