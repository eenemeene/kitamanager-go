/**
 * RFC 9457 problem documents, as the UI consumes them.
 *
 * Every API failure now arrives in this shape — including the ones gin used to
 * answer itself, which previously came back as plain text or an empty body.
 *
 * The reason this file exists rather than the UI reading `detail` directly is
 * language. The API speaks English and only English: `title` and `detail` are
 * English sentences written for whoever is reading a log or a curl output. A
 * German user should not see them. `code` is the stable, machine-readable member
 * that survives rewording on the server, so it is what the UI translates by, and
 * the catalogue below is the translation.
 */

import { defaultLocale, locales, type Locale } from '@/i18n/config';

/** One field a validation error rejected. */
export interface InvalidParam {
  field: string;
  reason: string;
}

/** The document the API returns for every error. */
export interface Problem {
  type?: string;
  title?: string;
  status?: number;
  detail?: string;
  instance?: string;
  code?: string;
  request_id?: string;
  invalid_params?: InvalidParam[];
}

/**
 * Error-code translations.
 *
 * These live here rather than in `src/i18n/messages/*.json` because
 * `getErrorMessage` is a plain function called from 54 places, most of them
 * outside a React render, so it cannot reach a `useTranslations` hook. Keeping
 * the catalogue beside the client also keeps it next to the codes it mirrors.
 *
 * A code with no entry falls through to the server's English detail, so adding
 * an error code on the backend degrades to today's behaviour rather than to a
 * blank message.
 */
const messages: Record<Locale, Record<string, string>> = {
  en: {
    not_found: 'That record no longer exists. It may have been deleted by someone else.',
    validation_error: 'Please correct the highlighted fields.',
    unauthorized: 'Your session has expired. Please sign in again.',
    forbidden: 'You do not have permission to do that.',
    too_many_requests: 'Too many requests. Please wait a moment and try again.',
    internal_error: 'Something went wrong on our side. Please try again.',
    method_not_allowed: 'That action is not available on this resource.',
    email_conflict: 'That email address is already in use.',
    contract_overlap:
      'The contract periods overlap. Adjust the dates so they do not cover the same day.',
    duplicate_bill_hash: 'This bill has already been imported.',
    duplicate_bill_month: 'A bill already exists for that month.',
    precondition_required:
      'Reload the page and try again — this change could not be checked against the current version.',
    precondition_failed:
      'Someone else changed this while you were editing. Reload to see their changes, then apply yours again.',
  },
  de: {
    not_found:
      'Dieser Datensatz existiert nicht mehr. Möglicherweise wurde er von jemand anderem gelöscht.',
    validation_error: 'Bitte korrigieren Sie die markierten Felder.',
    unauthorized: 'Ihre Sitzung ist abgelaufen. Bitte melden Sie sich erneut an.',
    forbidden: 'Sie haben keine Berechtigung für diese Aktion.',
    too_many_requests:
      'Zu viele Anfragen. Bitte warten Sie einen Moment und versuchen Sie es erneut.',
    internal_error: 'Auf unserer Seite ist ein Fehler aufgetreten. Bitte versuchen Sie es erneut.',
    method_not_allowed: 'Diese Aktion ist für diese Ressource nicht verfügbar.',
    email_conflict: 'Diese E-Mail-Adresse wird bereits verwendet.',
    contract_overlap:
      'Die Vertragszeiträume überschneiden sich. Bitte passen Sie die Daten so an, dass sie nicht denselben Tag abdecken.',
    duplicate_bill_hash: 'Diese Rechnung wurde bereits importiert.',
    duplicate_bill_month: 'Für diesen Monat existiert bereits eine Rechnung.',
    precondition_required:
      'Bitte laden Sie die Seite neu und versuchen Sie es erneut — diese Änderung konnte nicht gegen die aktuelle Version geprüft werden.',
    precondition_failed:
      'Jemand anderes hat diesen Datensatz während Ihrer Bearbeitung geändert. Bitte laden Sie neu und wenden Sie Ihre Änderung erneut an.',
  },
};

/**
 * The locale the user is reading in.
 *
 * next-intl resolves this server-side from the `locale` cookie; this reads the
 * same cookie, because the alternative — threading a locale through 54 call
 * sites — buys nothing. Outside a browser (tests, SSR) it is the default.
 */
export function currentLocale(): Locale {
  if (typeof document === 'undefined') {
    return defaultLocale;
  }
  const match = document.cookie.match(/(?:^|;\s*)locale=([^;]+)/);
  const value = match?.[1];
  return value && (locales as readonly string[]).includes(value)
    ? (value as Locale)
    : defaultLocale;
}

/** True when the value has the shape of a problem document. */
export function isProblem(data: unknown): data is Problem {
  if (typeof data !== 'object' || data === null) {
    return false;
  }
  const p = data as Problem;
  return typeof p.code === 'string' || typeof p.title === 'string';
}

/** Pulls the problem document out of an axios error, if there is one. */
export function getProblem(error: unknown): Problem | undefined {
  if (typeof error !== 'object' || error === null || !('response' in error)) {
    return undefined;
  }
  const data = (error as { response?: { data?: unknown } }).response?.data;
  return isProblem(data) ? data : undefined;
}

/** The fields a validation error rejected, for marking inputs in a form. */
export function getInvalidParams(error: unknown): InvalidParam[] {
  return getProblem(error)?.invalid_params ?? [];
}

/**
 * The request id, which is what support asks for when a user reports a 500.
 * Worth surfacing on internal errors and nowhere else — it is noise otherwise.
 */
export function getRequestId(error: unknown): string | undefined {
  return getProblem(error)?.request_id;
}

/**
 * The message to show a user for a problem document.
 *
 * The two locales resolve in opposite orders, deliberately:
 *
 *   - German prefers the translated code, because the server's `detail` is an
 *     English sentence and showing it is the bug this replaces. It falls back to
 *     `detail` only for codes with no translation, where an English sentence
 *     beats nothing.
 *   - English prefers `detail`, because it is the more specific of the two —
 *     it names the contract, the date, the field — and the translated string is
 *     the generic version of the same thing.
 */
export function problemMessage(problem: Problem): string | undefined {
  const catalogue = messages[currentLocale()];
  const translated = problem.code ? catalogue[problem.code] : undefined;

  if (currentLocale() === 'en') {
    return problem.detail || translated || problem.title || undefined;
  }
  return translated || problem.detail || problem.title || undefined;
}
