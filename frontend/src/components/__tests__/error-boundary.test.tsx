import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorBoundary } from '../error-boundary';

// A component that throws on demand, controlled via props.
function ThrowingChild({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error('Test explosion');
  }
  return <p>Child content</p>;
}

describe('ErrorBoundary', () => {
  // Suppress React's noisy error-boundary logging during these tests.
  let originalConsoleError: typeof console.error;

  beforeAll(() => {
    originalConsoleError = console.error;
    console.error = jest.fn();
  });

  afterAll(() => {
    console.error = originalConsoleError;
  });

  // -------------------------------------------------------------------
  // Happy path: children render normally
  // -------------------------------------------------------------------
  it('renders children when no error occurs', () => {
    render(
      <ErrorBoundary>
        <p>Hello world</p>
      </ErrorBoundary>
    );

    expect(screen.getByText('Hello world')).toBeInTheDocument();
  });

  // -------------------------------------------------------------------
  // Error caught: default fallback UI
  // -------------------------------------------------------------------
  it('shows default fallback UI when a child throws', () => {
    render(
      <ErrorBoundary>
        <ThrowingChild shouldThrow />
      </ErrorBoundary>
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('An unexpected error occurred')).toBeInTheDocument();
    expect(screen.getByText('Try Again')).toBeInTheDocument();
    // Original child must be gone
    expect(screen.queryByText('Child content')).not.toBeInTheDocument();
  });

  // -------------------------------------------------------------------
  // Custom title, message, retryLabel props
  // -------------------------------------------------------------------
  it('uses custom title prop in the fallback UI', () => {
    render(
      <ErrorBoundary title="Oops!">
        <ThrowingChild shouldThrow />
      </ErrorBoundary>
    );

    expect(screen.getByText('Oops!')).toBeInTheDocument();
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
  });

  it('uses custom message prop in the fallback UI', () => {
    render(
      <ErrorBoundary message="Please reload the page">
        <ThrowingChild shouldThrow />
      </ErrorBoundary>
    );

    expect(screen.getByText('Please reload the page')).toBeInTheDocument();
    expect(screen.queryByText('An unexpected error occurred')).not.toBeInTheDocument();
  });

  it('uses custom retryLabel prop on the retry button', () => {
    render(
      <ErrorBoundary retryLabel="Erneut versuchen">
        <ThrowingChild shouldThrow />
      </ErrorBoundary>
    );

    expect(screen.getByText('Erneut versuchen')).toBeInTheDocument();
    expect(screen.queryByText('Try Again')).not.toBeInTheDocument();
  });

  // -------------------------------------------------------------------
  // Custom fallback element replaces default UI entirely
  // -------------------------------------------------------------------
  it('renders custom fallback element instead of default card', () => {
    render(
      <ErrorBoundary fallback={<div>Custom fallback</div>}>
        <ThrowingChild shouldThrow />
      </ErrorBoundary>
    );

    expect(screen.getByText('Custom fallback')).toBeInTheDocument();
    // Default UI elements must not appear
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
    expect(screen.queryByText('Try Again')).not.toBeInTheDocument();
  });

  // -------------------------------------------------------------------
  // Retry button resets state and re-renders children
  // -------------------------------------------------------------------
  it('resets error state and re-renders children when retry is clicked', () => {
    // We need a component whose throwing behaviour can change between renders.
    // Use a ref so the value persists across the ErrorBoundary reset.
    let shouldThrow = true;

    function ConditionalThrower() {
      if (shouldThrow) {
        throw new Error('boom');
      }
      return <p>Recovered</p>;
    }

    render(
      <ErrorBoundary>
        <ConditionalThrower />
      </ErrorBoundary>
    );

    // Verify we are in error state
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();

    // Stop throwing, then click retry
    shouldThrow = false;
    fireEvent.click(screen.getByText('Try Again'));

    // Children should render again
    expect(screen.getByText('Recovered')).toBeInTheDocument();
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
  });

  // -------------------------------------------------------------------
  // componentDidCatch logs to console.error
  // -------------------------------------------------------------------
  it('calls console.error when an error is caught', () => {
    render(
      <ErrorBoundary>
        <ThrowingChild shouldThrow />
      </ErrorBoundary>
    );

    expect(console.error).toHaveBeenCalledWith(
      'ErrorBoundary caught:',
      expect.any(Error),
      expect.any(String)
    );
  });
});
