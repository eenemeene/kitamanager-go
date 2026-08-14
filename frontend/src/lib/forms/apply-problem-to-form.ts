import type {
  FieldValues,
  Path,
  UseFormClearErrors,
  UseFormGetValues,
  UseFormSetError,
} from 'react-hook-form';

import { getInvalidParams, type InvalidParam } from '@/lib/api/problem';

/**
 * Marks the fields a rejected submit named.
 *
 * The API reports validation failures as data — `invalid_params`, each with the
 * field's JSON path and a reason in the reader's language. Until now nothing
 * consumed it: a user who submitted three bad fields got one sentence and no
 * indication of which input to fix.
 *
 * # Nothing may vanish
 *
 * `setError` on a name react-hook-form does not know is a **silent no-op** — the
 * message disappears and the form looks like it accepted the value. That is the
 * failure mode this whole helper is shaped around: every violation either lands
 * on a field or is returned in `unmapped`, and the caller must show those. The
 * count of violations in equals the count marked plus the count returned.
 *
 * # Why a form may not have the field
 *
 * Three reasons, all real here:
 *
 *   - The form names it differently. Money is entered in euros and sent in
 *     cents (`amount_cents` against `entry_amount_euros`); a contract's dates are
 *     prefixed to disambiguate two date pairs on one form (`from` against
 *     `contract_from`). Those are the `aliases`.
 *   - The failure is about a collection rather than an input, like a bulk import
 *     reporting `add_children[3].contracts[1].from`, which no single control
 *     represents.
 *   - The endpoint validates something the form does not collect at all.
 */

export interface ApplyProblemResult {
  /** Violations that landed on a form field. */
  applied: number;
  /** Violations with nowhere to land. The caller must display these. */
  unmapped: InvalidParam[];
}

interface FormHandle<T extends FieldValues> {
  setError: UseFormSetError<T>;
  clearErrors: UseFormClearErrors<T>;
  getValues: UseFormGetValues<T>;
}

/** Resolves a dotted/indexed path against an object, as react-hook-form names it. */
function hasPath(values: unknown, path: string): boolean {
  const segments = path.replace(/\[(\d+)\]/g, '.$1').split('.');
  let node: unknown = values;
  for (const segment of segments) {
    if (node === null || typeof node !== 'object') {
      return false;
    }
    if (!(segment in (node as Record<string, unknown>))) {
      return false;
    }
    node = (node as Record<string, unknown>)[segment];
  }
  return true;
}

export function applyProblemToForm<T extends FieldValues>(
  error: unknown,
  form: FormHandle<T>,
  aliases: Record<string, string> = {}
): ApplyProblemResult {
  const violations = getInvalidParams(error);
  if (violations.length === 0) {
    return { applied: 0, unmapped: [] };
  }

  // Drop errors from a previous attempt first: a field the server complained
  // about last time may be fine now, and a stale message next to a corrected
  // value is worse than none.
  form.clearErrors();

  const values = form.getValues();
  const unmapped: InvalidParam[] = [];
  let applied = 0;
  let first: string | undefined;

  for (const violation of violations) {
    const name = aliases[violation.field] ?? violation.field;
    if (!hasPath(values, name)) {
      unmapped.push(violation);
      continue;
    }
    form.setError(
      name as Path<T>,
      {
        type: 'server',
        // The server sends the reader's language when the request negotiated
        // one, and English otherwise; `reason` is always present.
        message: violation.localized_reason || violation.reason,
      },
      // Focus the first one, so a long form scrolls to the problem rather than
      // leaving the user to hunt for it.
      first === undefined ? { shouldFocus: true } : undefined
    );
    if (first === undefined) {
      first = name;
    }
    applied += 1;
  }

  return { applied, unmapped };
}
