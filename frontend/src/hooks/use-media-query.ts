import { useSyncExternalStore } from 'react';

/**
 * Track a CSS media query match. Returns `false` during SSR and on the client's
 * first render, then reflects the real value after hydration. Uses
 * `useSyncExternalStore` so React can subscribe to the external matchMedia
 * source without violating the "no setState in effects" rule.
 */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (onChange) => {
      if (typeof window === 'undefined') return () => {};
      const mq = window.matchMedia(query);
      mq.addEventListener('change', onChange);
      return () => mq.removeEventListener('change', onChange);
    },
    () => {
      if (typeof window === 'undefined') return false;
      return window.matchMedia(query).matches;
    },
    () => false // SSR snapshot: mobile-first default
  );
}

/** Matches Tailwind's `lg:` breakpoint (≥1024px). */
export function useIsLgUp(): boolean {
  return useMediaQuery('(min-width: 1024px)');
}
