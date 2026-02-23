import {
  formatDate,
  formatDateForInput,
  formatDateForApi,
  calculateAge,
  formatCurrency,
  eurosToCents,
  centsToEuros,
  formatPeriod,
  formatFte,
  formatAgeRange,
  formatTime,
  combineDateAndTime,
  toLocalDateString,
  propertiesToValues,
  getPropertyValue,
  getScalarPropertyValue,
  setProperty,
  removePropertyByValue,
  hasPropertyValue,
  getKeyForValue,
  type ContractProperties,
} from '../formatting';

// ---------------------------------------------------------------------------
// formatDate
// ---------------------------------------------------------------------------
describe('formatDate', () => {
  it('returns dash for null', () => {
    expect(formatDate(null)).toBe('-');
  });

  it('returns dash for undefined', () => {
    expect(formatDate(undefined)).toBe('-');
  });

  it('formats a valid ISO date string in English locale', () => {
    const result = formatDate('2024-03-15', 'en');
    expect(result).toContain('Mar');
    expect(result).toContain('15');
    expect(result).toContain('2024');
  });

  it('formats a valid ISO date string in German locale', () => {
    const result = formatDate('2024-03-15', 'de');
    expect(result).toContain('März');
    expect(result).toContain('15');
    expect(result).toContain('2024');
  });

  it('returns the raw string for an invalid date', () => {
    expect(formatDate('not-a-date')).toBe('not-a-date');
  });

  it('defaults to English locale when none is specified', () => {
    const result = formatDate('2024-06-01');
    expect(result).toContain('Jun');
    expect(result).toContain('1');
    expect(result).toContain('2024');
  });
});

// ---------------------------------------------------------------------------
// formatDateForInput
// ---------------------------------------------------------------------------
describe('formatDateForInput', () => {
  it('returns empty string for null', () => {
    expect(formatDateForInput(null)).toBe('');
  });

  it('returns empty string for undefined', () => {
    expect(formatDateForInput(undefined)).toBe('');
  });

  it('formats a full ISO datetime to yyyy-MM-dd', () => {
    expect(formatDateForInput('2024-03-15T10:30:00Z')).toBe('2024-03-15');
  });

  it('formats a date-only string to yyyy-MM-dd', () => {
    expect(formatDateForInput('2024-07-04')).toBe('2024-07-04');
  });

  it('returns empty string for an invalid date', () => {
    expect(formatDateForInput('invalid')).toBe('');
  });
});

// ---------------------------------------------------------------------------
// formatDateForApi
// ---------------------------------------------------------------------------
describe('formatDateForApi', () => {
  it('returns null for null', () => {
    expect(formatDateForApi(null)).toBeNull();
  });

  it('returns null for undefined', () => {
    expect(formatDateForApi(undefined)).toBeNull();
  });

  it('returns the string as-is when it already contains T (RFC3339)', () => {
    const rfc = '2025-01-15T12:00:00Z';
    expect(formatDateForApi(rfc)).toBe(rfc);
  });

  it('appends T00:00:00Z to a YYYY-MM-DD string', () => {
    expect(formatDateForApi('2025-01-15')).toBe('2025-01-15T00:00:00Z');
  });

  it('returns null for empty string', () => {
    expect(formatDateForApi('')).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// calculateAge
// ---------------------------------------------------------------------------
describe('calculateAge', () => {
  it('calculates age for a valid birthdate', () => {
    const tenYearsAgo = new Date();
    tenYearsAgo.setFullYear(tenYearsAgo.getFullYear() - 10);
    const birthdate = toLocalDateString(tenYearsAgo);
    expect(calculateAge(birthdate)).toBe(10);
  });

  it('returns 0 for an invalid date string', () => {
    expect(calculateAge('invalid')).toBe(0);
  });

  it('returns 0 for an empty string', () => {
    expect(calculateAge('')).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// formatCurrency
// ---------------------------------------------------------------------------
describe('formatCurrency', () => {
  it('returns dash for null', () => {
    expect(formatCurrency(null)).toBe('-');
  });

  it('returns dash for undefined', () => {
    expect(formatCurrency(undefined)).toBe('-');
  });

  it('formats zero cents correctly', () => {
    const result = formatCurrency(0, 'en');
    expect(result).toContain('0.00');
    expect(result).toContain('€');
  });

  it('formats 166847 cents as 1,668.47 EUR in German locale', () => {
    const result = formatCurrency(166847, 'de');
    expect(result).toContain('1.668,47');
    expect(result).toContain('€');
  });

  it('formats 166847 cents as 1,668.47 EUR in English locale', () => {
    const result = formatCurrency(166847, 'en');
    expect(result).toContain('1,668.47');
    expect(result).toContain('€');
  });

  it('defaults to German locale', () => {
    const result = formatCurrency(100);
    expect(result).toContain('1,00');
    expect(result).toContain('€');
  });
});

// ---------------------------------------------------------------------------
// eurosToCents
// ---------------------------------------------------------------------------
describe('eurosToCents', () => {
  it('converts 1.50 euros to 150 cents', () => {
    expect(eurosToCents(1.5)).toBe(150);
  });

  it('converts 0 euros to 0 cents', () => {
    expect(eurosToCents(0)).toBe(0);
  });

  it('rounds 1.555 to 156 cents', () => {
    expect(eurosToCents(1.555)).toBe(156);
  });

  it('handles large amounts', () => {
    expect(eurosToCents(1668.47)).toBe(166847);
  });
});

// ---------------------------------------------------------------------------
// centsToEuros
// ---------------------------------------------------------------------------
describe('centsToEuros', () => {
  it('converts 150 cents to 1.50 euros', () => {
    expect(centsToEuros(150)).toBe(1.5);
  });

  it('converts 0 cents to 0 euros', () => {
    expect(centsToEuros(0)).toBe(0);
  });

  it('handles large amounts', () => {
    expect(centsToEuros(166847)).toBe(1668.47);
  });
});

// ---------------------------------------------------------------------------
// formatPeriod
// ---------------------------------------------------------------------------
describe('formatPeriod', () => {
  it('formats a period with both dates', () => {
    const result = formatPeriod('2024-01-01', '2024-12-31', 'en');
    expect(result).toContain('Jan');
    expect(result).toContain('Dec');
    expect(result).toContain(' - ');
  });

  it('shows ongoing text when end date is null', () => {
    const result = formatPeriod('2024-01-01', null, 'en');
    expect(result).toContain('ongoing');
  });

  it('shows ongoing text when end date is undefined', () => {
    const result = formatPeriod('2024-01-01', undefined, 'en');
    expect(result).toContain('ongoing');
  });

  it('uses custom ongoing text', () => {
    const result = formatPeriod('2024-01-01', null, 'en', 'present');
    expect(result).toContain('present');
  });
});

// ---------------------------------------------------------------------------
// formatFte
// ---------------------------------------------------------------------------
describe('formatFte', () => {
  it('formats integer ratio to two decimals', () => {
    expect(formatFte(1)).toBe('1.00');
  });

  it('formats fractional ratio to two decimals', () => {
    expect(formatFte(0.5)).toBe('0.50');
    expect(formatFte(0.75)).toBe('0.75');
  });

  it('truncates beyond two decimals', () => {
    expect(formatFte(0.333)).toBe('0.33');
  });
});

// ---------------------------------------------------------------------------
// formatAgeRange
// ---------------------------------------------------------------------------
describe('formatAgeRange', () => {
  it('returns dash when both values are null', () => {
    expect(formatAgeRange(null, null)).toBe('-');
  });

  it('formats max-only range with null min', () => {
    expect(formatAgeRange(null, 3, 'years')).toBe('< 3 years');
  });

  it('formats max-only range with undefined min', () => {
    expect(formatAgeRange(undefined, 3, 'years')).toBe('< 3 years');
  });

  it('formats min-only range with null max', () => {
    expect(formatAgeRange(3, null, 'years')).toBe('3+ years');
  });

  it('formats min-only range with undefined max', () => {
    expect(formatAgeRange(3, undefined, 'years')).toBe('3+ years');
  });

  it('formats full range', () => {
    expect(formatAgeRange(3, 6, 'years')).toBe('3-6 years');
  });

  it('accepts translated text', () => {
    expect(formatAgeRange(3, 6, 'Jahre')).toBe('3-6 Jahre');
    expect(formatAgeRange(null, 3, 'Jahre')).toBe('< 3 Jahre');
    expect(formatAgeRange(3, null, 'Jahre')).toBe('3+ Jahre');
  });
});

// ---------------------------------------------------------------------------
// propertiesToValues
// ---------------------------------------------------------------------------
describe('propertiesToValues', () => {
  it('returns empty array for undefined', () => {
    expect(propertiesToValues(undefined)).toEqual([]);
  });

  it('returns empty array for empty object', () => {
    expect(propertiesToValues({})).toEqual([]);
  });

  it('extracts scalar string values', () => {
    const props: ContractProperties = { care_type: 'fulltime', meal: 'lunch' };
    const result = propertiesToValues(props);
    expect(result).toEqual(expect.arrayContaining(['fulltime', 'lunch']));
    expect(result).toHaveLength(2);
  });

  it('extracts array values by spreading them', () => {
    const props: ContractProperties = { extras: ['music', 'sports'] };
    const result = propertiesToValues(props);
    expect(result).toEqual(expect.arrayContaining(['music', 'sports']));
    expect(result).toHaveLength(2);
  });

  it('handles mixed scalar and array values', () => {
    const props: ContractProperties = {
      care_type: 'fulltime',
      extras: ['music', 'sports'],
    };
    const result = propertiesToValues(props);
    expect(result).toEqual(expect.arrayContaining(['fulltime', 'music', 'sports']));
    expect(result).toHaveLength(3);
  });
});

// ---------------------------------------------------------------------------
// getPropertyValue
// ---------------------------------------------------------------------------
describe('getPropertyValue', () => {
  it('returns the value for an existing key', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    expect(getPropertyValue(props, 'care_type')).toBe('fulltime');
  });

  it('returns undefined for a missing key', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    expect(getPropertyValue(props, 'missing')).toBeUndefined();
  });

  it('returns undefined for undefined properties', () => {
    expect(getPropertyValue(undefined, 'care_type')).toBeUndefined();
  });

  it('returns array values as-is', () => {
    const props: ContractProperties = { extras: ['music', 'sports'] };
    expect(getPropertyValue(props, 'extras')).toEqual(['music', 'sports']);
  });
});

// ---------------------------------------------------------------------------
// getScalarPropertyValue
// ---------------------------------------------------------------------------
describe('getScalarPropertyValue', () => {
  it('returns string value for a scalar property', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    expect(getScalarPropertyValue(props, 'care_type')).toBe('fulltime');
  });

  it('returns undefined when value is an array', () => {
    const props: ContractProperties = { extras: ['music', 'sports'] };
    expect(getScalarPropertyValue(props, 'extras')).toBeUndefined();
  });

  it('returns undefined for undefined properties', () => {
    expect(getScalarPropertyValue(undefined, 'care_type')).toBeUndefined();
  });

  it('returns undefined for a missing key', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    expect(getScalarPropertyValue(props, 'missing')).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// setProperty
// ---------------------------------------------------------------------------
describe('setProperty', () => {
  it('sets a new key on undefined properties', () => {
    const result = setProperty(undefined, 'care_type', 'fulltime');
    expect(result).toEqual({ care_type: 'fulltime' });
  });

  it('sets a new key on existing properties', () => {
    const props: ContractProperties = { meal: 'lunch' };
    const result = setProperty(props, 'care_type', 'fulltime');
    expect(result).toEqual({ meal: 'lunch', care_type: 'fulltime' });
  });

  it('overwrites an existing key', () => {
    const props: ContractProperties = { care_type: 'parttime' };
    const result = setProperty(props, 'care_type', 'fulltime');
    expect(result).toEqual({ care_type: 'fulltime' });
  });

  it('does not mutate the original properties', () => {
    const props: ContractProperties = { meal: 'lunch' };
    setProperty(props, 'care_type', 'fulltime');
    expect(props).toEqual({ meal: 'lunch' });
  });

  it('supports array values', () => {
    const result = setProperty(undefined, 'extras', ['music', 'sports']);
    expect(result).toEqual({ extras: ['music', 'sports'] });
  });
});

// ---------------------------------------------------------------------------
// removePropertyByValue
// ---------------------------------------------------------------------------
describe('removePropertyByValue', () => {
  it('returns undefined for undefined properties', () => {
    expect(removePropertyByValue(undefined, 'fulltime')).toBeUndefined();
  });

  it('removes the entry with matching value', () => {
    const props: ContractProperties = { care_type: 'fulltime', meal: 'lunch' };
    const result = removePropertyByValue(props, 'fulltime');
    expect(result).toEqual({ meal: 'lunch' });
  });

  it('returns undefined when removing the last property', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    const result = removePropertyByValue(props, 'fulltime');
    expect(result).toBeUndefined();
  });

  it('returns all properties unchanged when value is not found', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    const result = removePropertyByValue(props, 'parttime');
    expect(result).toEqual({ care_type: 'fulltime' });
  });

  it('does not mutate the original properties', () => {
    const props: ContractProperties = { care_type: 'fulltime', meal: 'lunch' };
    removePropertyByValue(props, 'fulltime');
    expect(props).toEqual({ care_type: 'fulltime', meal: 'lunch' });
  });
});

// ---------------------------------------------------------------------------
// hasPropertyValue
// ---------------------------------------------------------------------------
describe('hasPropertyValue', () => {
  it('returns true when value exists', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    expect(hasPropertyValue(props, 'fulltime')).toBe(true);
  });

  it('returns false when value does not exist', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    expect(hasPropertyValue(props, 'parttime')).toBe(false);
  });

  it('returns false for undefined properties', () => {
    expect(hasPropertyValue(undefined, 'fulltime')).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// getKeyForValue
// ---------------------------------------------------------------------------
describe('getKeyForValue', () => {
  it('returns the key when value is found', () => {
    const props: ContractProperties = { care_type: 'fulltime', meal: 'lunch' };
    expect(getKeyForValue(props, 'fulltime')).toBe('care_type');
  });

  it('returns undefined when value is not found', () => {
    const props: ContractProperties = { care_type: 'fulltime' };
    expect(getKeyForValue(props, 'parttime')).toBeUndefined();
  });

  it('returns undefined for undefined properties', () => {
    expect(getKeyForValue(undefined, 'fulltime')).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// formatTime
// ---------------------------------------------------------------------------
describe('formatTime', () => {
  it('returns empty string for null', () => {
    expect(formatTime(null)).toBe('');
  });

  it('returns empty string for undefined', () => {
    expect(formatTime(undefined)).toBe('');
  });

  it('formats an ISO datetime to HH:mm in local time', () => {
    // Parse the UTC string and verify it displays in local time
    const date = new Date('2025-06-15T08:30:00Z');
    const expected = `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
    expect(formatTime('2025-06-15T08:30:00Z')).toBe(expected);
  });

  it('formats a datetime with different time', () => {
    const date = new Date('2025-06-15T16:45:00Z');
    const expected = `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
    expect(formatTime('2025-06-15T16:45:00Z')).toBe(expected);
  });

  it('returns empty string for invalid input', () => {
    expect(formatTime('not-a-date')).toBe('');
  });
});

// ---------------------------------------------------------------------------
// combineDateAndTime
// ---------------------------------------------------------------------------
describe('combineDateAndTime', () => {
  it('returns null when time is empty', () => {
    expect(combineDateAndTime('2025-06-15', '')).toBeNull();
  });

  it('combines date and time into a valid ISO string', () => {
    const result = combineDateAndTime('2025-06-15', '08:30');
    expect(result).not.toBeNull();
    // The result should be a valid ISO string that, when parsed back to local time,
    // gives the original date and time
    const parsed = new Date(result!);
    expect(parsed.getFullYear()).toBe(2025);
    expect(parsed.getMonth()).toBe(5); // June = 5
    expect(parsed.getDate()).toBe(15);
    expect(parsed.getHours()).toBe(8);
    expect(parsed.getMinutes()).toBe(30);
  });

  it('round-trips with formatTime', () => {
    const result = combineDateAndTime('2025-12-01', '16:00');
    expect(result).not.toBeNull();
    expect(formatTime(result!)).toBe('16:00');
  });

  it('returns null for invalid date', () => {
    expect(combineDateAndTime('invalid', '08:30')).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// toLocalDateString
// ---------------------------------------------------------------------------
describe('toLocalDateString', () => {
  it('formats a date to YYYY-MM-DD', () => {
    expect(toLocalDateString(new Date(2026, 0, 5))).toBe('2026-01-05');
  });

  it('pads single-digit month and day', () => {
    expect(toLocalDateString(new Date(2025, 2, 3))).toBe('2025-03-03');
  });

  it('handles Dec 31', () => {
    expect(toLocalDateString(new Date(2025, 11, 31))).toBe('2025-12-31');
  });

  it('handles Jan 1', () => {
    expect(toLocalDateString(new Date(2026, 0, 1))).toBe('2026-01-01');
  });

  it('uses local timezone, not UTC', () => {
    // A UTC time of 23:30 on Feb 23 is Feb 24 in CET (UTC+1)
    // This test is timezone-dependent: it verifies that toLocalDateString
    // uses local getters, so the result matches the local date.
    const date = new Date('2026-02-23T23:30:00Z');
    const result = toLocalDateString(date);
    // The result should match the local date, not the UTC date
    const expected = `${date.getFullYear()}-${(date.getMonth() + 1).toString().padStart(2, '0')}-${date.getDate().toString().padStart(2, '0')}`;
    expect(result).toBe(expected);
  });

  it('round-trips formatTime(combineDateAndTime(date, time)) preserves time', () => {
    const dateStr = '2026-06-15';
    const timeStr = '14:30';
    const combined = combineDateAndTime(dateStr, timeStr);
    expect(combined).not.toBeNull();
    expect(formatTime(combined!)).toBe(timeStr);
  });
});
