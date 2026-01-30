import {
  formatDate,
  formatDateForInput,
  calculateAge,
  formatCurrency,
  eurosToCents,
  centsToEuros,
  formatPeriod,
  formatFte,
  formatAgeRange,
} from '../formatting';

describe('formatDate', () => {
  it('returns dash for null or undefined', () => {
    expect(formatDate(null)).toBe('-');
    expect(formatDate(undefined)).toBe('-');
  });

  it('formats a valid ISO date string', () => {
    const result = formatDate('2024-03-15', 'en');
    expect(result).toContain('Mar');
    expect(result).toContain('15');
    expect(result).toContain('2024');
  });

  it('formats date in German locale', () => {
    const result = formatDate('2024-03-15', 'de');
    expect(result).toContain('März');
    expect(result).toContain('15');
    expect(result).toContain('2024');
  });

  it('returns original string for invalid date', () => {
    expect(formatDate('not-a-date')).toBe('not-a-date');
  });
});

describe('formatDateForInput', () => {
  it('returns empty string for null or undefined', () => {
    expect(formatDateForInput(null)).toBe('');
    expect(formatDateForInput(undefined)).toBe('');
  });

  it('formats date as YYYY-MM-DD', () => {
    expect(formatDateForInput('2024-03-15T10:30:00Z')).toBe('2024-03-15');
  });

  it('returns empty string for invalid date', () => {
    expect(formatDateForInput('invalid')).toBe('');
  });
});

describe('calculateAge', () => {
  it('calculates age correctly', () => {
    const tenYearsAgo = new Date();
    tenYearsAgo.setFullYear(tenYearsAgo.getFullYear() - 10);
    const birthdate = tenYearsAgo.toISOString().split('T')[0];

    expect(calculateAge(birthdate)).toBe(10);
  });

  it('returns 0 for invalid date', () => {
    expect(calculateAge('invalid')).toBe(0);
  });
});

describe('formatCurrency', () => {
  it('returns dash for null or undefined', () => {
    expect(formatCurrency(null)).toBe('-');
    expect(formatCurrency(undefined)).toBe('-');
  });

  it('formats cents as EUR in German locale', () => {
    const result = formatCurrency(166847, 'de');
    expect(result).toContain('1.668,47');
    expect(result).toContain('€');
  });

  it('formats cents as EUR in English locale', () => {
    const result = formatCurrency(166847, 'en');
    expect(result).toContain('1,668.47');
    expect(result).toContain('€');
  });

  it('handles zero correctly', () => {
    const result = formatCurrency(0, 'de');
    expect(result).toContain('0,00');
    expect(result).toContain('€');
  });
});

describe('eurosToCents', () => {
  it('converts euros to cents', () => {
    expect(eurosToCents(10.5)).toBe(1050);
    expect(eurosToCents(1668.47)).toBe(166847);
    expect(eurosToCents(0)).toBe(0);
  });

  it('rounds correctly', () => {
    expect(eurosToCents(10.999)).toBe(1100);
    expect(eurosToCents(10.001)).toBe(1000);
  });
});

describe('centsToEuros', () => {
  it('converts cents to euros', () => {
    expect(centsToEuros(1050)).toBe(10.5);
    expect(centsToEuros(166847)).toBe(1668.47);
    expect(centsToEuros(0)).toBe(0);
  });
});

describe('formatPeriod', () => {
  it('formats a period with both dates', () => {
    const result = formatPeriod('2024-01-01', '2024-12-31', 'en');
    expect(result).toContain('Jan');
    expect(result).toContain('Dec');
    expect(result).toContain('-');
  });

  it('formats ongoing period with custom text', () => {
    const result = formatPeriod('2024-01-01', null, 'en', 'present');
    expect(result).toContain('Jan');
    expect(result).toContain('present');
  });

  it('uses default ongoing text', () => {
    const result = formatPeriod('2024-01-01', undefined, 'en');
    expect(result).toContain('ongoing');
  });
});

describe('formatFte', () => {
  it('formats FTE with two decimal places', () => {
    expect(formatFte(1)).toBe('1.00');
    expect(formatFte(0.5)).toBe('0.50');
    expect(formatFte(0.75)).toBe('0.75');
    expect(formatFte(0.333)).toBe('0.33');
  });
});

describe('formatAgeRange', () => {
  it('returns dash when both values are null', () => {
    expect(formatAgeRange(null, null)).toBe('-');
  });

  it('formats max-only range', () => {
    expect(formatAgeRange(null, 3, 'en')).toBe('< 3 years');
    expect(formatAgeRange(undefined, 3, 'en')).toBe('< 3 years');
  });

  it('formats min-only range', () => {
    expect(formatAgeRange(3, null, 'en')).toBe('3+ years');
    expect(formatAgeRange(3, undefined, 'en')).toBe('3+ years');
  });

  it('formats full range', () => {
    expect(formatAgeRange(3, 6, 'en')).toBe('3-6 years');
  });

  it('formats in German', () => {
    expect(formatAgeRange(3, 6, 'de')).toBe('3-6 Jahre');
    expect(formatAgeRange(null, 3, 'de')).toBe('< 3 Jahre');
    expect(formatAgeRange(3, null, 'de')).toBe('3+ Jahre');
  });
});
