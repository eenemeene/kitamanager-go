import React from 'react';
import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from '@/components/ui/tooltip';

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

// The providers go in via RTL's `wrapper` rather than around `ui`, because
// `wrapper` is what the returned `rerender` keeps. Wrapping `ui` inline meant
// `rerender(<Thing />)` replaced the whole tree, providers included, and the
// component threw "No QueryClient set" on the second render -- which makes any
// test that drives a prop through a change (a dialog's `open`, say) look like a
// component bug.
export function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createTestQueryClient();
  function Providers({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>{children}</TooltipProvider>
      </QueryClientProvider>
    );
  }
  return render(ui, { wrapper: Providers });
}

export function createHookWrapper(queryClient?: QueryClient) {
  const qc = queryClient ?? createTestQueryClient();
  function TestWrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: qc }, children);
  }
  return TestWrapper;
}

export function createMockPaginatedResponse<T>(data: T[], overrides = {}) {
  return { data, total: data.length, page: 1, limit: 30, total_pages: 1, ...overrides };
}
