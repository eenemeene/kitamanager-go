/**
 * Proof that formatting follows the reader's language.
 *
 * The bug this hook exists to fix was not that either language was wrong on its
 * own — it was that the two halves disagreed. `formatCurrency` defaulted to
 * German and `formatDate` to English, and almost nobody passed a locale, so a
 * German card printed a German euro amount beside an English date and an
 * English card did the reverse. A test that only checks one locale cannot see
 * that; these run the same call twice and require the outputs to differ.
 */

import { renderHook } from '@testing-library/react';

import { useFormatters } from '../use-formatters';

let mockLocale = 'en';
jest.mock('next-intl', () => ({
  useLocale: () => mockLocale,
  useTranslations: () => (key: string) => key,
}));

function formattersFor(locale: string) {
  mockLocale = locale;
  return renderHook(() => useFormatters()).result.current;
}

/**
 * `Intl` separates a German amount from its currency sign with a non-breaking
 * space, which is correct output and invisible in a failure diff. Normalising
 * keeps these assertions readable.
 */
const plain = (s: string) => s.replace(/ /g, ' ');

describe('useFormatters', () => {
  describe('follows the chosen locale', () => {
    it('formats currency in the reader’s convention', () => {
      expect(plain(formattersFor('de').currency(166847))).toBe('1.668,47 €');
      expect(plain(formattersFor('en').currency(166847))).toBe('€1,668.47');
    });

    it('formats dates in the reader’s convention', () => {
      expect(formattersFor('de').date('2026-03-15')).toBe('15. März 2026');
      expect(formattersFor('en').date('2026-03-15')).toBe('Mar 15, 2026');
    });

    it('formats month labels in the reader’s convention', () => {
      expect(formattersFor('de').monthYear('2026-03-01')).toBe('März 26');
      expect(formattersFor('en').monthYear('2026-03-01')).toBe('Mar 26');
    });

    it('formats decimals with the reader’s separator', () => {
      expect(formattersFor('de').fte(1.5)).toBe('1,50');
      expect(formattersFor('en').fte(1.5)).toBe('1.50');
      expect(formattersFor('de').percentage(12.5)).toBe('12,50%');
      expect(formattersFor('en').percentage(12.5)).toBe('12.50%');
    });
  });

  describe('is internally consistent', () => {
    // The actual defect: money and dates rendered from different locales in the
    // same breath. Whatever language is chosen, both have to be that language.
    it.each(['de', 'en'])('renders money and dates in the same language (%s)', (locale) => {
      const fmt = formattersFor(locale);
      const money = plain(fmt.currency(166847));
      const date = fmt.date('2026-03-15');

      if (locale === 'de') {
        expect(money).toContain('1.668,47');
        expect(date).toContain('März');
      } else {
        expect(money).toContain('1,668.47');
        expect(date).toContain('Mar 15');
      }
    });
  });

  it('exposes the locale and its Intl tag', () => {
    expect(formattersFor('de')).toMatchObject({ locale: 'de', intl: 'de-DE' });
    expect(formattersFor('en')).toMatchObject({ locale: 'en', intl: 'en-US' });
  });

  it('handles an open-ended period with the caller’s wording', () => {
    expect(plain(formattersFor('de').period('2026-01-01', null, 'laufend'))).toBe(
      '1. Jan. 2026 - laufend'
    );
  });

  it('passes null amounts through as a dash rather than formatting them', () => {
    expect(formattersFor('de').currency(null)).toBe('-');
    expect(formattersFor('de').date(null)).toBe('-');
  });
});
