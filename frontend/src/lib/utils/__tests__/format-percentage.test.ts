import { formatPercentage } from '../formatting';

describe('formatPercentage', () => {
  it('formats a pre-multiplied value with two fraction digits (de locale)', () => {
    expect(formatPercentage(12.5, 'de')).toBe('12,50%');
  });

  it('formats a fractional value when asFraction=true', () => {
    expect(formatPercentage(0.125, 'de', 2, true)).toBe('12,50%');
  });

  it('honors fractionDigits', () => {
    expect(formatPercentage(33.333333, 'de', 1)).toBe('33,3%');
  });

  it('uses en-US separators when locale=en', () => {
    expect(formatPercentage(1234.5, 'en', 2)).toBe('1,234.50%');
  });

  it('returns - for null/undefined/non-finite', () => {
    expect(formatPercentage(null, 'de')).toBe('-');
    expect(formatPercentage(undefined, 'de')).toBe('-');
    expect(formatPercentage(NaN, 'de')).toBe('-');
    expect(formatPercentage(Infinity, 'de')).toBe('-');
  });
});
