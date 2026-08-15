'use client';

import { useEffect, useRef } from 'react';
import { useTranslations } from 'next-intl';
import { AlertCircle } from 'lucide-react';
import type { FieldErrors, FieldValues } from 'react-hook-form';

import { Alert } from '@/components/ui/alert';
import type { InvalidParam } from '@/lib/api/problem';

/**
 * Lists everything wrong with a submitted form, at the top of it.
 *
 * Marking the fields alone is not enough: these forms live in dialogs capped at
 * `max-h-[85vh]` with their own scrollbar, and on a phone the submit button sits
 * at the bottom of that scroll. A user who presses it lands on a marked field
 * they cannot see. The summary is what tells them how many things are wrong and
 * takes them to each one.
 *
 * # Why focus lands here and not on the first field
 *
 * Focusing the first invalid input is the obvious alternative and is worse on
 * touch: it summons the on-screen keyboard, which takes roughly 40% of the
 * viewport on a phone and a comparable slice of a tablet in portrait, and can
 * hide the very field it just focused. This container is a div with
 * `tabIndex={-1}`, so focusing it moves the screen reader and the scroll
 * position without opening anything.
 *
 * # Scrolling
 *
 * `scrollIntoView` walks up to the nearest scrollable ancestor on its own, which
 * matters because roughly half these forms are inside a dialog's scroll
 * container and the rest are on the page — `window.scrollTo` would do nothing
 * for the first group. Items centre rather than top-align, because on a phone
 * the keyboard occupies the lower half once focus lands on the input.
 */

export interface FormErrorSummaryItem {
  /** The form field, when there is one to jump to. */
  name?: string;
  /** What the user reads: the field's label followed by the reason. */
  message: string;
}

interface FormErrorSummaryProps<T extends FieldValues> {
  /** react-hook-form's errors — client-side rules and server violations alike. */
  errors: FieldErrors<T>;
  /** Server violations with no field on this form. Must still be shown. */
  unmapped?: InvalidParam[];
  /**
   * Field name to the label the form shows beside it. Without an entry the
   * field name is used, which is ugly but never silent — and a test asserts
   * every field a form can report has one.
   */
  labels?: Record<string, string>;
}

function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  );
}

export function FormErrorSummary<T extends FieldValues>({
  errors,
  unmapped = [],
  labels = {},
}: FormErrorSummaryProps<T>) {
  const t = useTranslations();
  const container = useRef<HTMLDivElement>(null);

  const fieldItems: FormErrorSummaryItem[] = Object.entries(errors).flatMap(([name, error]) => {
    const message = (error as { message?: string } | undefined)?.message;
    if (!message) {
      return [];
    }
    const label = labels[name] ?? name;
    // "Vorname ist erforderlich" — the label in the nominative followed by the
    // reason as a predicate, which is how both catalogues phrase them.
    return [{ name, message: `${label} ${message}` }];
  });

  const unmappedItems: FormErrorSummaryItem[] = unmapped.map((violation) => ({
    // No `name`: there is no input to jump to. The field path is still shown,
    // because "something is invalid" with no locator is worse than a technical
    // string the user can report.
    message: `${labels[violation.field] ?? violation.field} ${
      violation.localized_reason || violation.reason
    }`,
  }));

  const items = [...fieldItems, ...unmappedItems];
  const count = items.length;

  useEffect(() => {
    if (count > 0) {
      container.current?.focus();
    }
  }, [count]);

  if (count === 0) {
    return null;
  }

  const jumpTo = (name: string) => {
    const field = document.getElementById(name);
    if (!field) {
      return;
    }
    // Focus first without scrolling, then scroll deliberately: letting focus do
    // the scrolling top-aligns the field under the keyboard on a phone.
    field.focus({ preventScroll: true });
    field.scrollIntoView({
      behavior: prefersReducedMotion() ? 'auto' : 'smooth',
      block: 'center',
    });
  };

  return (
    <Alert
      ref={container}
      variant="destructive"
      tabIndex={-1}
      data-testid="form-error-summary"
      // The count is on the element as well as in the heading, because the
      // heading is a translated ICU plural — a test asserting its text would be
      // asserting the message catalogue, not this component.
      data-count={count}
      className="mb-4 outline-none"
    >
      <div className="flex items-start gap-2">
        <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
        <div className="min-w-0 flex-1">
          <p className="font-medium">{t('forms.errorSummary.title', { count })}</p>
          <ul className="mt-2 space-y-1">
            {items.map((item, index) => (
              <li key={item.name ?? `unmapped-${index}`}>
                {item.name ? (
                  <button
                    type="button"
                    onClick={() => jumpTo(item.name as string)}
                    // 44px minimum, because these are tap targets on a phone and
                    // a tablet, not just links on a desktop.
                    className="flex min-h-11 w-full items-center text-left underline underline-offset-4"
                  >
                    {item.message}
                  </button>
                ) : (
                  <span className="flex min-h-11 items-center">{item.message}</span>
                )}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </Alert>
  );
}
