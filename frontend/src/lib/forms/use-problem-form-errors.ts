'use client';

import { useEffect, useState } from 'react';
import type {
  FieldValues,
  UseFormClearErrors,
  UseFormGetValues,
  UseFormSetError,
} from 'react-hook-form';

import type { InvalidParam } from '@/lib/api/problem';
import { applyProblemToForm } from './apply-problem-to-form';

/**
 * Applies a rejected submit to the form that produced it.
 *
 * The dialogs own their `useForm` while the mutation lives in the page that
 * renders them, so the error arrives as a prop rather than through a callback.
 * react-query keeps the last error on the mutation, which makes this a matter of
 * reacting to a value instead of threading a handler down.
 *
 * Pair it with the page suppressing its own toast when the error carries field
 * violations — the page can tell without touching the form, via
 * `getInvalidParams(error).length > 0`. Keeping those two decisions apart is
 * what stops the dialog needing to reach back up into the page.
 */
export function useProblemFormErrors<T extends FieldValues>(
  error: unknown,
  form: {
    setError: UseFormSetError<T>;
    clearErrors: UseFormClearErrors<T>;
    getValues: UseFormGetValues<T>;
  },
  aliases?: Record<string, string>
): InvalidParam[] {
  const [unmapped, setUnmapped] = useState<InvalidParam[]>([]);

  useEffect(() => {
    if (!error) {
      // The mutation reset, or a retry succeeded: drop the previous attempt's
      // collection-level problems rather than leaving them on screen next to a
      // form that is now fine.
      setUnmapped([]);
      return;
    }
    const { unmapped: left } = applyProblemToForm(error, form, aliases);
    setUnmapped(left);
    // `form` is a set of stable react-hook-form callbacks and `aliases` is a
    // literal; keying on the error alone is what makes this run once per
    // rejection instead of on every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [error]);

  return unmapped;
}
