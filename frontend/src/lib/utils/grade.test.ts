import { compareGrade } from './grade';

describe('compareGrade', () => {
  it('orders TVöD-SuE grades naturally', () => {
    const got = ['S10', 'S2', 'S11a', 'S11b', 'S9', 'S8a', 'S8b', 'Minijob'];
    got.sort(compareGrade);
    expect(got).toEqual(['Minijob', 'S2', 'S8a', 'S8b', 'S9', 'S10', 'S11a', 'S11b']);
  });

  it.each<[string, string, 'lt' | 'gt' | 'eq']>([
    // Numeric component beats alphabetic ordering on equal prefix
    ['S2', 'S10', 'lt'],
    ['S10', 'S2', 'gt'],
    // Same number, suffix orders alphabetically
    ['S8a', 'S8b', 'lt'],
    ['S8b', 'S8a', 'gt'],
    ['S8a', 'S8a', 'eq'],
    // Different prefix sorts on prefix first
    ['Minijob', 'S2', 'lt'],
    // Strings without digits still total-order
    ['foo', 'bar', 'gt'],
    ['bar', 'bar', 'eq'],
  ])('compareGrade(%s, %s) is %s', (a, b, want) => {
    const got = compareGrade(a, b);
    if (want === 'lt') expect(got).toBeLessThan(0);
    if (want === 'gt') expect(got).toBeGreaterThan(0);
    if (want === 'eq') expect(got).toBe(0);
  });
});
