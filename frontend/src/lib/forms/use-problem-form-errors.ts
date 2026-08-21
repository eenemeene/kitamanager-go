'use client';

import { useEffect, useRef, useState } from 'react';
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
  // dialog, most often. The one to apply is whichever *just* failed, and that
  // is a question about what changed since the last render, not about the
  // array's order.
  //
  // Reading it as order ("the first truthy one") is what this used to do, and
  // it fails in an ordinary sequence: a rejected create leaves its error on the
  // mutation forever -- react-query keeps it until the next attempt or an
  // explicit reset, and nothing here resets -- so a later rejected *update*
  // never became "first truthy". The effect then did not re-run at all, because
  // the value it was keyed on had not changed. The update's violations were
  // dropped, and `suppressesToast` still saw a rejection carrying
  // `invalid_params` and swallowed the toast. A failed submit with nothing on
  // screen: no marked fields, no summary, no toast.
  //
  // So: remember what each slot held last render, and act on a slot that has
  // just taken a *new* rejection. Resubmitting the same form badly twice
  // re-applies and re-focuses, because react-query builds a fresh error object
  // per attempt.
  const errors = Array.isArray(error) ? error : [error];
  const previousErrors = useRef<unknown[]>([]);

  // No dependency array: the effect itself decides whether anything happened,
  // by comparing against the ref. Keying it on a derived value is what broke
  // before -- there is no single value whose identity tracks "any of these
  // mutations just failed", and `errors` is a fresh array on every render, so
  // listing it would be the same thing written less honestly.
  //
  // exhaustive-deps warns that calling setUnmapped here could loop. It cannot:
  // the only unconditional call passes a function that returns the *same* array
  // when there is nothing to clear, which react bails out of. The
  // "does not re-apply on an unrelated re-render" test holds that line.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    const previous = previousErrors.current;
    previousErrors.current = errors;

    // The newest rejection: a slot holding something it was not holding
    // before. Later slots win a tie, which only arises if two mutations reject
    // between one render and the next.
    let latest: unknown;
    for (let i = 0; i < errors.length; i++) {
      if (errors[i] && errors[i] !== previous[i]) {
        latest = errors[i];
      }
    }

    if (latest) {
      const { unmapped: left } = applyProblemToForm(latest, form, aliases);
      setUnmapped(left);
      return;
    }

    if (!errors.some(Boolean)) {
      // Every mutation reset, or a retry succeeded: drop the previous attempt's
      // collection-level problems rather than leaving them on screen next to a
      // form that is now fine. Same array when already empty, so this cannot
      // re-render in a loop.
      setUnmapped((current) => (current.length === 0 ? current : []));
    }
  });

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
