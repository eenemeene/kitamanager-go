'use client';

import { useEffect, useRef } from 'react';

/** Anything holding a rejection that can be dropped -- a react-query mutation. */
export interface Resettable {
  reset: () => void;
}

/**
 * Drops pending rejections when a dialog opens, so a form never opens showing
 * the previous attempt's errors.
 *
 * react-query keeps a mutation's error until the next attempt or an explicit
 * reset, and closing a dialog is neither. `useCrudDialogs` resets the *form*,
 * which clears the marked fields -- but the collection-level violations that
 * had no field to land on live in `useProblemFormErrors`, keyed off the
 * mutation error that is still sitting there. Reopening the dialog therefore
 * re-displayed a summary belonging to a submit the user had already abandoned,
 * often for a different record entirely.
 *
 * Resetting on open rather than on close is deliberate: a rejected submit
 * leaves the dialog open, and clearing on close would also have to survive the
 * close-on-success path. "A dialog that just opened has nothing pending" is the
 * whole rule.
 *
 * @param open      whether the dialog is currently open
 * @param resettables the mutations that submit this dialog
 */
export function useResetOnReopen(open: boolean, ...resettables: Array<Resettable | undefined>) {
  // The mutation objects are rebuilt every render, so they cannot be effect
  // dependencies -- the effect would fire on every render and reset a rejection
  // the moment it arrived. The ref keeps the latest ones reachable while the
  // effect stays keyed on the only thing that should trigger it.
  const latest = useRef(resettables);

  // Written in an effect rather than during render: a ref is not part of the
  // render output, and mutating one while rendering is what makes a component
  // disagree with what it just drew. This effect has no dependency list, so it
  // runs after every commit and the ref is current by the time the one below
  // can fire.
  useEffect(() => {
    latest.current = resettables;
  });

  useEffect(() => {
    if (!open) return;
    for (const resettable of latest.current) {
      resettable?.reset();
    }
  }, [open]);
}
