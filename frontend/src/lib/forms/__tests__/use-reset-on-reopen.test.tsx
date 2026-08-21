import { renderHook } from '@testing-library/react';
import { useResetOnReopen } from '../use-reset-on-reopen';

function mutation() {
  return { reset: jest.fn() };
}

describe('useResetOnReopen', () => {
  it('resets everything the moment the dialog opens', () => {
    const create = mutation();
    const update = mutation();
    const { rerender } = renderHook(
      ({ open }: { open: boolean }) => useResetOnReopen(open, create, update),
      { initialProps: { open: false } }
    );
    expect(create.reset).not.toHaveBeenCalled();

    rerender({ open: true });

    expect(create.reset).toHaveBeenCalledTimes(1);
    expect(update.reset).toHaveBeenCalledTimes(1);
  });

  it('resets again on the next open, not just the first', () => {
    const create = mutation();
    const { rerender } = renderHook(
      ({ open }: { open: boolean }) => useResetOnReopen(open, create),
      {
        initialProps: { open: true },
      }
    );
    expect(create.reset).toHaveBeenCalledTimes(1);

    rerender({ open: false });
    rerender({ open: true });

    expect(create.reset).toHaveBeenCalledTimes(2);
  });

  it('leaves a rejection alone while the dialog stays open', () => {
    // The critical one: a rejected submit keeps the dialog open and re-renders
    // it. If this fired on every render it would wipe the error the user needs
    // to read, and the summary would flash and vanish.
    const create = mutation();
    const { rerender } = renderHook(
      ({ open }: { open: boolean }) => useResetOnReopen(open, create),
      {
        initialProps: { open: true },
      }
    );
    expect(create.reset).toHaveBeenCalledTimes(1);

    rerender({ open: true });
    rerender({ open: true });

    expect(create.reset).toHaveBeenCalledTimes(1);
  });

  it('reaches the current mutation objects, not the ones from first render', () => {
    // react-query rebuilds the result object every render; the reset that runs
    // must be the live one.
    const first = mutation();
    const second = mutation();
    const { rerender } = renderHook(
      ({ open, m }: { open: boolean; m: { reset: jest.Mock } }) => useResetOnReopen(open, m),
      { initialProps: { open: false, m: first } }
    );

    rerender({ open: true, m: second });

    expect(first.reset).not.toHaveBeenCalled();
    expect(second.reset).toHaveBeenCalledTimes(1);
  });

  it('tolerates an absent mutation', () => {
    const create = mutation();
    expect(() => renderHook(() => useResetOnReopen(true, create, undefined))).not.toThrow();
    expect(create.reset).toHaveBeenCalledTimes(1);
  });
});
