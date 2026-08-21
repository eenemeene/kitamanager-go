import type { QueryClient } from '@tanstack/react-query';

/**
 * Imperative access to the app's one QueryClient, so that ending a session can
 * drop everything it cached.
 *
 * The client is created in `Providers` and lives for the life of the tab. Both
 * logout paths are outside React: `useAuthStore.logout` is a zustand action, and
 * the 401 handler is registered on the api client at module scope. Neither can
 * hold a hook, and neither should own a second QueryClient — hence a registry
 * rather than an import of the instance itself, which would also make the
 * client a module singleton shared across SSR requests.
 *
 * # Why the cache has to go
 *
 * Logout and login are both soft navigations, so a user switch inside one tab
 * never remounts the provider. Everything the previous user read stays cached
 * and is served instantly to the next one, because `staleTime` is a minute.
 *
 * Most keys are organization-scoped, which limits the damage to data the new
 * user is probably entitled to anyway. Two are not: the settings page reads
 * `['factors', 'me']` and `['me', 'sessions']`, and both are about *a person* —
 * their enrolled MFA factors, and their active sessions with IP addresses, user
 * agents and last-seen times. Scoping those keys by user id is defence in depth
 * and is done as well; clearing the cache is the fix, because it does not
 * require every future key to remember to carry an identity.
 */
let queryClient: QueryClient | undefined;

/** Called once by `Providers` with the client it created. */
export function registerQueryClient(client: QueryClient) {
  queryClient = client;
}

/** Test seam: forget the registered client. */
export function unregisterQueryClient() {
  queryClient = undefined;
}

/**
 * Drop every cached query and mutation.
 *
 * Call when a session ends — an explicit logout, or a 401 telling us the server
 * has already ended it. `clear()` rather than `invalidateQueries()`: invalidated
 * data is still handed to components while the refetch runs, which is exactly
 * the window this closes.
 */
export function clearSessionCache() {
  queryClient?.clear();
}
