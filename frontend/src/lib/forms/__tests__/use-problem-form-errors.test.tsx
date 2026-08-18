import { renderHook, waitFor } from '@testing-library/react';
import { useProblemFormErrors, suppressesToast } from '../use-problem-form-errors';

/** A rejection shaped like the API's problem document. */
function rejection(params: Array<{ field: string; reason: string }>) {
  return {
    response: {
      data: {
        type: 'u',
        title: 'Validation failed',
        status: 400,
        code: 'validation',
        invalid_params: params.map((p) => ({ rule: 'required', ...p })),
      },
    },
  };
}

function form(values: Record<string, unknown> = { name: '' }) {
  return {
    setError: jest.fn(),
    clearErrors: jest.fn(),
    getValues: jest.fn(() => values),
  };
}

/**
 * Rejections are hoisted out of the render callback on purpose.
 *
 * The hook keys its effect on the error's identity, so that submitting the same
 * bad form twice re-applies the marking and re-focuses the first field. That
 * means a caller handing it a freshly-built object on every render would loop --
 * react-query never does, it keeps one error object until the next attempt, but
 * a test written the obvious way does, and it takes the whole worker with it.
 */
describe('useProblemFormErrors', () => {
  it('marks the field the server named', async () => {
    const f = form();
    const error = rejection([{ field: 'name', reason: 'is required' }]);
    renderHook(() => useProblemFormErrors(error, f as never));

    await waitFor(() => expect(f.setError).toHaveBeenCalled());
    expect(f.setError.mock.calls[0][0]).toBe('name');
  });

  it('keeps a violation it cannot place, rather than dropping it', async () => {
    // The form has no such input, so nothing can be marked -- but the user still
    // has to be told, or the submit fails for a reason nobody states.
    const f = form();
    const error = rejection([{ field: 'not_on_this_form', reason: 'is invalid' }]);
    const { result } = renderHook(() => useProblemFormErrors(error, f as never));

    await waitFor(() => expect(result.current).toHaveLength(1));
    expect(result.current[0].field).toBe('not_on_this_form');
    expect(f.setError).not.toHaveBeenCalled();
  });

  it('does nothing when the rejection names no field', async () => {
    const f = form();
    const error = new Error('network down');
    const { result } = renderHook(() => useProblemFormErrors(error, f as never));

    await waitFor(() => expect(result.current).toHaveLength(0));
    expect(f.setError).not.toHaveBeenCalled();
  });

  it('clears what it reported once the error goes away', async () => {
    // A corrected form must not keep showing the previous attempt's problems.
    const f = form();
    const { result, rerender } = renderHook(
      ({ error }) => useProblemFormErrors(error, f as never),
      { initialProps: { error: rejection([{ field: 'gone', reason: 'is invalid' }]) as unknown } }
    );

    await waitFor(() => expect(result.current).toHaveLength(1));
    rerender({ error: undefined });
    await waitFor(() => expect(result.current).toHaveLength(0));
  });

  it('applies whichever of several mutations failed', async () => {
    // One dialog, two mutations: create and edit. Passing both is what lets a
    // single summary serve the form regardless of which submit was rejected.
    const f = form();
    const error = rejection([{ field: 'name', reason: 'is already taken' }]);
    renderHook(() => useProblemFormErrors([undefined, error], f as never));

    await waitFor(() => expect(f.setError).toHaveBeenCalled());
    expect(f.setError.mock.calls[0][0]).toBe('name');
  });
});

describe('suppressesToast', () => {
  it('is true when the server named fields, because the summary shows them', () => {
    expect(suppressesToast(rejection([{ field: 'name', reason: 'is required' }]))).toBe(true);
  });

  it('is true even when no field could be placed, because the summary still shows them', () => {
    expect(suppressesToast(rejection([{ field: 'unknown', reason: 'is invalid' }]))).toBe(true);
  });

  it('is false for a rejection with no fields, where the toast is the only report', () => {
    // A conflict or a network failure has nothing to mark, and silence would
    // leave a rejected submit looking like nothing happened.
    expect(suppressesToast(new Error('network down'))).toBe(false);
  });
});
