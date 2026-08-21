'use client';

import { useMemo } from 'react';
import { useLocale } from 'next-intl';

import type { Locale } from '@/i18n/config';
import {
  formatCurrency,
  formatDate,
  formatFte,
  formatMonthYear,
  formatNumber,
  formatPercentage,
  formatPeriod,
  intlLocale,
} from '@/lib/utils/formatting';

/**
 * The display formatters, bound to the language the reader chose.
 *
 * Before this, formatting was the one part of the UI that did not follow the
 * language switch. The shared helpers took an optional locale that almost
 * nobody passed, twenty-one files hardcoded 'de-DE' or 'en-US' inline, and the
 * two defaults pointed in opposite directions — so a German user read English
 * dates beside German euro amounts, and an English user read the reverse.
 *
 * Reading `useLocale()` here, once, is what makes that unforgettable: the
 * underlying functions now require a locale, and this is the only convenient
 * way to supply one. A component that wants a formatted value has to come
 * through here, and once it does it is correct in both languages for free.
 *
 * `locale` and `intl` are exposed for the handful of places that need to build
 * their own `Intl` formatter — a chart tooltip with unusual options, say —
 * rather than reaching for a literal tag.
 */
export interface Formatters {
  /** Cents to a euro amount: "1.234,56 €" / "€1,234.56". */
  currency: (cents: number | null | undefined) => string;
  /** An ISO date to a medium-length date: "20. Aug. 2026" / "Aug 20, 2026". */
  date: (dateString: string | null | undefined) => string;
  /** A from/to range, with `ongoingText` standing in for an open end. */
  period: (from: string, to: string | null | undefined, ongoingText?: string) => string;
  /** Month and year, for chart axes and table headers: "Aug 26". */
  monthYear: (date: string | Date, options?: Intl.DateTimeFormatOptions) => string;
  /** A number with the locale's separators. */
  number: (value: number, options?: Intl.NumberFormatOptions) => string;
  /** A percentage, from a pre-multiplied value unless `asFraction`. */
  percentage: (
    value: number | null | undefined,
    fractionDigits?: number,
    asFraction?: boolean
  ) => string;
  /** A full-time-equivalent ratio to two decimals. */
  fte: (ratio: number) => string;
  /** The app's locale code, 'de' or 'en'. */
  locale: Locale;
  /** The BCP 47 tag for `Intl`, 'de-DE' or 'en-US'. */
  intl: string;
}

export function useFormatters(): Formatters {
  const locale = useLocale() as Locale;

  return useMemo(
    () => ({
      currency: (cents) => formatCurrency(cents, locale),
      date: (dateString) => formatDate(dateString, locale),
      period: (from, to, ongoingText) => formatPeriod(from, to, locale, ongoingText),
      monthYear: (date, options) => formatMonthYear(date, locale, options),
      number: (value, options) => formatNumber(value, locale, options),
      percentage: (value, fractionDigits, asFraction) =>
        formatPercentage(value, locale, fractionDigits, asFraction),
      fte: (ratio) => formatFte(ratio, locale),
      locale,
      intl: intlLocale(locale),
    }),
    [locale]
  );
}
