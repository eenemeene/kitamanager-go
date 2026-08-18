'use client';

import { useEffect, useState } from 'react';
import type {
  FieldValues,
  UseFormClearErrors,
  UseFormGetValues,
  UseFormSetError,
} from 'react-hook-form';

import { getInvalidParams, type InvalidParam } from '@/lib/api/problem';
import { applyProblemToForm } from './apply-problem-to-form';

/**
 * The slice of react-hook-form needed to mark a rejected field. Taking the three
 * callbacks rather than the whole form object keeps the dependency narrow and
 * matches what applyProblemToForm expects.
 */
export interface ProblemFormTarget<T extends FieldValues> {
  setError: UseFormSetError<T>;
  clearErrors: UseFormClearErrors<T>;
  getValues: UseFormGetValues<T>;
}

/**
 * Applies a rejected submit to the form that produced it.
 *
 * This is the only way field violations reach a form. It works by watching the
 * mutation's error rather than by hooking into the mutation, which is what lets
 * it serve every mutation hook in the codebase without any of them knowing that
 * forms exist -- react-query keeps the last error, so a rejection is a value to
 * react to rather than a callback to thread.
 *
 * Pass every mutation whose rejections belong to this form. A page with separate
 * create and edit forms passes each form only the mutation that submits it,
 * otherwise a rejected edit marks fields on the create form.
 *
 * Pair it with `suppressesToast` on the mutation hook, so the toast does not
 * repeat what the summary already lists. Keeping those two decisions apart is
 * what stops a form needing to reach into the hook that submits it.
 */
export function useProblemFormErrors<T extends FieldValues>(
  error: unknown | unknown[],
  form: ProblemFormTarget<T>,
  aliases?: Record<string, string>
): InvalidParam[] {
  const [unmapped, setUnmapped] = useState<InvalidParam[]>([]);

  // Several mutations can feed one form -- create and update on the same
  // dialog, most often. Whichever most recently failed is the one to apply.
  const errors = Array.isArray(error) ? error : [error];
  const active = errors.find(Boolean);

  useEffect(() => {
    if (!active) {
      // The mutation reset, or a retry succeeded: drop the previous attempt's
      // collection-level problems rather than leaving them on screen next to a
      // form that is now fine.
      setUnmapped([]);
      return;
    }
    const { unmapped: left } = applyProblemToForm(active, form, aliases);
    setUnmapped(left);
    // `form` is a set of stable react-hook-form callbacks and `aliases` is a
    // literal; keying on the error alone is what makes this run once per
    // rejection instead of on every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active]);

  return unmapped;
}

/**
 * Whether a rejection is already accounted for by a form, and so should not also
 * raise a toast.
 *
 * Hand it to a mutation hook's `onMutationError`. It answers from the problem
 * document alone -- if the server named fields, the form's summary is showing
 * them, mapped or not. A conflict or a network failure names none, and there the
 * toast is the last thing between a rejected submit and silence.
 */
export function suppressesToast(error: unknown): boolean {
  return getInvalidParams(error).length > 0;
}
