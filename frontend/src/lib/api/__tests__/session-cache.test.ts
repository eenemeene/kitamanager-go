import { QueryClient } from '@tanstack/react-query';
import { clearSessionCache, registerQueryClient, unregisterQueryClient } from '../session-cache';

describe('session cache', () => {
  afterEach(() => unregisterQueryClient());

  it('drops cached data for the registered client', () => {
    // The leak this closes: logout and login are both soft navigations, so the
    // QueryClient outlives the account switch and hands the next user whatever
    // the previous one read.
    const client = new QueryClient();
    client.setQueryData(['factors', 1], { factors: [{ id: 1, type: 'totp' }] });
    client.setQueryData(['sessions', 1], { sessions: [{ id: 'a', ip: '203.0.113.7' }] });
    registerQueryClient(client);

    clearSessionCache();

    expect(client.getQueryData(['factors', 1])).toBeUndefined();
    expect(client.getQueryData(['sessions', 1])).toBeUndefined();
  });

  it('removes rather than invalidates, so nothing is served while a refetch runs', () => {
    // invalidateQueries would leave the data readable until the refetch lands,
    // which is exactly the window that leaked.
    const client = new QueryClient();
    client.setQueryData(['children', 3, 'list'], [{ id: 1, first_name: 'Emma' }]);
    registerQueryClient(client);

    clearSessionCache();

    expect(client.getQueryCache().getAll()).toHaveLength(0);
  });

  it('is a no-op when no client has been registered', () => {
    // Server-side rendering, and any test that imports the auth store without
    // mounting Providers.
    expect(() => clearSessionCache()).not.toThrow();
  });

  it('forgets the client once unregistered', () => {
    const client = new QueryClient();
    client.setQueryData(['a'], 1);
    registerQueryClient(client);
    unregisterQueryClient();

    clearSessionCache();

    expect(client.getQueryData(['a'])).toBe(1);
  });

  it('clears the most recently registered client', () => {
    const first = new QueryClient();
    const second = new QueryClient();
    first.setQueryData(['a'], 1);
    second.setQueryData(['b'], 2);
    registerQueryClient(first);
    registerQueryClient(second);

    clearSessionCache();

    expect(second.getQueryData(['b'])).toBeUndefined();
  });
});
