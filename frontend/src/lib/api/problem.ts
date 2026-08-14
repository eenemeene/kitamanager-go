/**
 * RFC 9457 problem documents, as the UI consumes them.
 *
 * Every API failure arrives in this shape — including the ones gin used to
 * answer itself, which previously came back as plain text or an empty body.
 *
 * Language is handled by the server. The top-level `title` and `detail` are
 * always English, for logs, captured responses and whoever picks up the support
 * ticket; a `localized` block carries the same thing in the language the request
 * asked for, negotiated via `Accept-Language`. The UI shows `localized` when it
 * is present and English when it is not, and holds no catalogue of its own.
 */

import { defaultLocale, locales, type Locale } from '@/i18n/config';

/**
 * The locale the user chose in the app, read from the cookie next-intl uses.
 *
 * Sent as `Accept-Language` so the server can localize. Deliberately not the
 * browser's own header: a German user on an English-locale laptop has already
 * told us which they want, in the app.
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

/** One field a validation error rejected. */
export interface InvalidParam {
  field: string;
  /** English sentence fragment, for a client that does not localize. */
  reason: string;
  /** The validator rule: "required", "email", "min", "max", "voucher". */
  rule: string;
  /** The rule's argument, where it takes one. */
  param?: string;
  /** `reason` in the negotiated language, on the same terms as `localized`. */
  localized_reason?: string;
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
  /** The specifics as data, for a client that renders its own message. */
  params?: Record<string, string>;
  /** The user-facing view, present when the request negotiated a language the
   *  server has a catalogue for. The members above stay English. */
  localized?: { locale: string; title?: string; detail?: string };
  invalid_params?: InvalidParam[];
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
 * The server localizes. It sends the English `title`/`detail` for logs and
 * captured responses, and a `localized` block in the language the request asked
 * for — so this picks the localized text when it is there and the English when
 * it is not.
 *
 * This used to be a translation layer: a catalogue keyed by error code, a set of
 * codes whose message was self-contained, per-rule sentences for field errors,
 * and a rule that appended the English detail in parentheses so a German reader
 * was never told less than an English one. All of it existed because the server
 * could not say it in German. It can now — every user-facing message is
 * registered and translated, enforced by a test — so the layer is gone rather
 * than kept as a fallback that would quietly rot.
 */
export function problemMessage(problem: Problem): string | undefined {
  return problem.localized?.detail || problem.detail || problem.localized?.title || problem.title;
}
