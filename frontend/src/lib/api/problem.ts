/**
 * RFC 9457 problem documents, as the UI consumes them.
 *
 * Every API failure now arrives in this shape — including the ones gin used to
 * answer itself, which previously came back as plain text or an empty body.
 *
 * The reason this file exists rather than the UI reading `detail` directly is
 * language. The API speaks English and only English: `title` and `detail` are
 * English sentences written for whoever is reading a log or a curl output. `code`
 * is the stable, machine-readable member that survives rewording on the server,
 * so it is what the UI translates by, and the catalogue below is the translation.
 *
 * The constraint that shapes everything here: **translating must not cost the
 * reader information.** German is this product's primary language, and a
 * translated sentence is necessarily more generic than a `detail` built for one
 * occurrence — so where the detail carries specifics the template cannot express,
 * both are shown. See `problemMessage` for the exact order, and `selfContained`
 * for the codes where the translation genuinely says it all.
 */

import { defaultLocale, locales, type Locale } from '@/i18n/config';

/** One field a validation error rejected. */
export interface InvalidParam {
  field: string;
  /** English sentence fragment, for a client that does not localize. */
  reason: string;
  /** The validator rule: "required", "email", "min", "max", "voucher". */
  rule: string;
  /** The rule's argument, where it takes one. */
  param?: string;
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
  /** The specifics as data, for interpolating into a localized message. */
  params?: Record<string, string>;
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
    // Used instead of the above when Retry-After gave us a number of seconds.
    too_many_requests_retry: 'Too many requests. Please try again in {seconds} seconds.',
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
    too_many_requests_retry:
      'Zu viele Anfragen. Bitte versuchen Sie es in {seconds} Sekunden erneut.',
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
 * Codes whose translated sentence says everything the server's `detail` says.
 *
 * For these, `detail` is generic on the server too — `contract_overlap` is the
 * bare sentinel "period would overlap with existing record", `internal_error` is
 * fixed text — so the translation loses nothing and appending the English would
 * be noise.
 *
 * Every other code's `detail` is the *only* place the specifics live: which
 * child, which index in a bulk import, which field. Around 205 `BadRequest`
 * sites build details like `add_children[3].contracts[1]: from is required`, and
 * none of that survives a generic translation. Those codes get the detail shown
 * as well — see `problemMessage`.
 *
 * A code belongs here only once its message genuinely carries everything, which
 * for most means once the server sends `params` for it.
 */
const selfContained = new Set([
  'contract_overlap',
  'precondition_required',
  'precondition_failed',
  'internal_error',
  'unauthorized',
  'forbidden',
  'too_many_requests',
  'method_not_allowed',
]);

/** Fills `{name}` placeholders from the problem's params. */
function interpolate(template: string, params: Record<string, string> | undefined): string {
  if (!params) {
    return template;
  }
  return template.replace(/\{(\w+)\}/g, (whole, key: string) => params[key] ?? whole);
}

/**
 * Renders a validation failure in the reader's language.
 *
 * `invalid_params` carries the validator's `rule` and its argument, not just the
 * English `reason`, which is what makes a fully German sentence possible: the
 * field name is data, the rule is an enum, so nothing has to be recovered from
 * English prose. The output is as specific as the English `detail` — same fields,
 * same order — which is the whole point.
 */
function validationMessage(problem: Problem, locale: Locale): string | undefined {
  const params = problem.invalid_params;
  if (!params?.length) {
    return undefined;
  }
  const rules = fieldRules[locale];
  return params
    .map((p) => {
      const template = rules[p.rule] ?? rules.default;
      return interpolate(template, { field: p.field, param: p.param ?? '' });
    })
    .join('; ');
}

/** Per-rule sentence fragments for a rejected field. */
const fieldRules: Record<Locale, Record<string, string>> = {
  en: {
    required: '{field} is required',
    email: '{field} must be a valid email address',
    min: '{field} must be at least {param} characters',
    max: '{field} must be at most {param} characters',
    voucher: '{field} must match the voucher format GB-XXXXXXXXXXX-NN',
    default: '{field} is invalid',
  },
  de: {
    required: '{field} ist erforderlich',
    email: '{field} muss eine gültige E-Mail-Adresse sein',
    min: '{field} muss mindestens {param} Zeichen lang sein',
    max: '{field} darf höchstens {param} Zeichen lang sein',
    voucher: '{field} muss dem Gutscheinformat GB-XXXXXXXXXXX-NN entsprechen',
    default: '{field} ist ungültig',
  },
};

/**
 * The message to show a user for a problem document.
 *
 * The rule that matters: **a German reader is never told less than an English
 * one.** An earlier version of this preferred the translated sentence over
 * `detail` outright, which read well and quietly dropped the specifics — the
 * field that was rejected, the row of the import that failed — because those live
 * only in the English `detail`. German is this product's primary language; it
 * does not get the abridged edition.
 *
 * So, in order:
 *
 *   1. A validation failure renders from `invalid_params`, fully translated and
 *      exactly as specific as the English sentence.
 *   2. A code with a translation gets it, with any `params` interpolated.
 *   3. If that translation is not `selfContained`, the server's `detail` is
 *      appended, because it holds specifics the template cannot express yet.
 *   4. No translation at all falls through to `detail` — an English sentence
 *      beats a blank toast.
 *
 * English resolves the same way; step 3 simply never appends, since the
 * translation and the detail are then the same language and the detail wins.
 */
export function problemMessage(problem: Problem): string | undefined {
  const locale = currentLocale();

  const validation = validationMessage(problem, locale);
  if (validation) {
    return validation;
  }

  // A 429 that told us how long to wait uses the variant that says so. Without
  // this the seconds would reach an English reader through `detail` and be lost
  // on a German one, since too_many_requests is self-contained.
  const key =
    problem.code === 'too_many_requests' && problem.params?.seconds
      ? 'too_many_requests_retry'
      : problem.code;

  const template = key ? messages[locale][key] : undefined;
  if (!template) {
    return problem.detail || problem.title || undefined;
  }

  const translated = interpolate(template, problem.params);
  if (locale === 'en') {
    return problem.detail || translated;
  }
  if (problem.detail && !(problem.code && selfContained.has(problem.code))) {
    return `${translated} (${problem.detail})`;
  }
  return translated;
}
