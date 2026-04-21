import '@testing-library/jest-dom';
import 'jest-axe/extend-expect';

// Mock next-intl
jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key) => key;
    t.has = () => false;
    return t;
  },
  useLocale: () => 'en',
}));

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
    back: jest.fn(),
    prefetch: jest.fn(),
  }),
  usePathname: () => '/',
  useSearchParams: () => new URLSearchParams(),
}));

// Browser-only mocks (skip in Node test environment)
if (typeof window !== 'undefined') {
  // Mock window.matchMedia. Default to matching for min-width queries at and
  // below the `lg` (1024px) breakpoint so components render in their
  // desktop/lg state by default. Tests that want to exercise tablet or phone
  // behavior can override per-test.
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: jest.fn().mockImplementation((query) => {
      const minWidthMatch = query.match(/min-width:\s*(\d+)px/);
      const minWidthPx = minWidthMatch ? parseInt(minWidthMatch[1], 10) : 0;
      return {
        matches: minWidthPx <= 1024,
        media: query,
        onchange: null,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn(),
      };
    }),
  });
}

// Mock ResizeObserver
if (typeof global.ResizeObserver === 'undefined') {
  global.ResizeObserver = jest.fn().mockImplementation(() => ({
    observe: jest.fn(),
    unobserve: jest.fn(),
    disconnect: jest.fn(),
  }));
}
