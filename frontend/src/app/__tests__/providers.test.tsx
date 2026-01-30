import { render, screen } from '@testing-library/react';
import { useQueryClient, QueryClient } from '@tanstack/react-query';
import { Providers } from '../providers';

// Create a test component that uses QueryClient
function QueryClientConsumer() {
  const queryClient = useQueryClient();
  return (
    <span data-testid="query-client">{queryClient ? 'has-query-client' : 'no-query-client'}</span>
  );
}

describe('Providers', () => {
  it('renders children', () => {
    render(
      <Providers>
        <div data-testid="child">Test Child</div>
      </Providers>
    );

    expect(screen.getByTestId('child')).toBeInTheDocument();
    expect(screen.getByText('Test Child')).toBeInTheDocument();
  });

  it('provides QueryClient context', () => {
    render(
      <Providers>
        <QueryClientConsumer />
      </Providers>
    );

    expect(screen.getByTestId('query-client')).toHaveTextContent('has-query-client');
  });

  it('configures QueryClient with correct defaults', () => {
    let capturedClient: QueryClient | null = null;

    function ClientCapture() {
      capturedClient = useQueryClient();
      return null;
    }

    render(
      <Providers>
        <ClientCapture />
      </Providers>
    );

    expect(capturedClient).not.toBeNull();
    const defaultOptions = capturedClient!.getDefaultOptions();
    expect(defaultOptions?.queries?.staleTime).toBe(60 * 1000);
    expect(defaultOptions?.queries?.retry).toBe(1);
  });

  it('renders multiple children', () => {
    render(
      <Providers>
        <div data-testid="child-1">Child 1</div>
        <div data-testid="child-2">Child 2</div>
      </Providers>
    );

    expect(screen.getByTestId('child-1')).toBeInTheDocument();
    expect(screen.getByTestId('child-2')).toBeInTheDocument();
  });
});
