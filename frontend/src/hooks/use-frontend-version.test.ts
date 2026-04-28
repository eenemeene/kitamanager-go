/**
 * @jest-environment jsdom
 */
import { renderHook } from '@testing-library/react';
import { useFrontendVersion } from './use-frontend-version';

describe('useFrontendVersion', () => {
  afterEach(() => {
    document.querySelectorAll('meta[name="kitamanager-version"]').forEach((m) => m.remove());
  });

  it('returns the meta tag content when present', () => {
    const meta = document.createElement('meta');
    meta.setAttribute('name', 'kitamanager-version');
    meta.setAttribute('content', 'v0.29.0');
    document.head.appendChild(meta);

    const { result } = renderHook(() => useFrontendVersion());
    expect(result.current).toBe('v0.29.0');
  });

  it('returns empty string when the meta tag is missing', () => {
    // Old layout / cached SSR output without the meta tag — the hook
    // distinguishes "not yet read" (null) from "read, no value" (empty
    // string) so the sidebar can hide the row in the latter case.
    const { result } = renderHook(() => useFrontendVersion());
    expect(result.current).toBe('');
  });

  it('returns empty string when the meta tag has no content attribute', () => {
    const meta = document.createElement('meta');
    meta.setAttribute('name', 'kitamanager-version');
    document.head.appendChild(meta);

    const { result } = renderHook(() => useFrontendVersion());
    expect(result.current).toBe('');
  });
});
