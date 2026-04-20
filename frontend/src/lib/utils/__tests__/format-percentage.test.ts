import { formatPercentage } from '../formatting';

describe('formatPercentage', () => {
  it('formats a pre-multiplied value with two fraction digits (de locale)', () => {
    expect(formatPercentage(12.5)).toBe('12,50%');
  });

  it('formats a fractional value when asFraction=true', () => {
    expect(formatPercentage(0.125, 2, 'de', true)).toBe('12,50%');
  });

  it('honors fractionDigits', () => {
    expect(formatPercentage(33.333333, 1)).toBe('33,3%');
  });

  it('uses en-US separators when locale=en', () => {
    expect(formatPercentage(1234.5, 2, 'en')).toBe('1,234.50%');
  });

  it('returns - for null/undefined/non-finite', () => {
    expect(formatPercentage(null)).toBe('-');
    expect(formatPercentage(undefined)).toBe('-');
    expect(formatPercentage(NaN)).toBe('-');
    expect(formatPercentage(Infinity)).toBe('-');
  });
});
