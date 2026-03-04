import { locales, defaultLocale, localeNames, type Locale } from '../config';

describe('i18n config', () => {
  it('exports en and de locales', () => {
    expect(locales).toContain('en');
    expect(locales).toContain('de');
    expect(locales).toHaveLength(2);
  });

  it('defaults to English', () => {
    expect(defaultLocale).toBe('en');
  });

  it('has display names for all locales', () => {
    for (const locale of locales) {
      expect(localeNames[locale]).toBeDefined();
      expect(localeNames[locale].length).toBeGreaterThan(0);
    }
  });

  it('has correct display names', () => {
    expect(localeNames.en).toBe('English');
    expect(localeNames.de).toBe('Deutsch');
  });

  it('Locale type accepts valid locales', () => {
    const valid: Locale = 'en';
    expect(locales).toContain(valid);
  });
});
