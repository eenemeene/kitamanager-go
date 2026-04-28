import { useSyncExternalStore } from 'react';

/**
 * Reads the frontend's own version from the `<meta name="kitamanager-version">`
 * tag injected by the root layout. The meta tag is populated at build time from
 * APP_VERSION (Dockerfile.frontend's builder stage), so this is the version of
 * the bundle currently running in the browser — not necessarily the same as the
 * API's version (the two ship as separate images).
 *
 * Lives in the DOM rather than via a separate HTTP route because:
 *   - the report-pdf tool already reads the same meta tag via Playwright;
 *     using one source means UI and PDFs can never disagree
 *   - no extra network call to keep in sync
 *   - works even if the backend is unreachable
 *
 * Implemented with useSyncExternalStore so React handles the SSR-vs-client
 * snapshot pairing properly (no hydration mismatch warning) and so we don't
 * trigger the `react-hooks/set-state-in-effect` lint rule that the simpler
 * useState+useEffect shape would. The meta tag never mutates after the
 * initial render, so `subscribe` returns a no-op unsubscriber.
 */
export function useFrontendVersion(): string {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

const subscribe = () => () => {};

function getSnapshot(): string {
  return document.querySelector('meta[name="kitamanager-version"]')?.getAttribute('content') ?? '';
}

function getServerSnapshot(): string {
  return '';
}
