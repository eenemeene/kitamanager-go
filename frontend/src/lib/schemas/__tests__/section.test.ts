import { sectionSchema } from '../section';

// ---------------------------------------------------------------------------
// optionalAge preprocessor (tested indirectly via sectionSchema fields)
// ---------------------------------------------------------------------------
describe('optionalAge preprocessor', () => {
  it('converts empty string to null (field absent from result)', () => {
    const result = sectionSchema.parse({ name: 'Toddlers', min_age_months: '' });
    expect(result.min_age_months).toBeNull();
  });

  it('converts null to null', () => {
    const result = sectionSchema.parse({
      name: 'Toddlers',
      min_age_months: null,
    });
    expect(result.min_age_months).toBeNull();
  });

  it('converts undefined to undefined (field omitted)', () => {
    const result = sectionSchema.parse({
      name: 'Toddlers',
      min_age_months: undefined,
    });
    expect(result.min_age_months).toBeUndefined();
  });

  it('converts NaN to null', () => {
    const result = sectionSchema.parse({ name: 'Toddlers', min_age_months: NaN });
    expect(result.min_age_months).toBeNull();
  });

  it('converts a non-numeric string to null', () => {
    const result = sectionSchema.parse({ name: 'Toddlers', min_age_months: 'abc' });
    expect(result.min_age_months).toBeNull();
  });

  it('accepts a valid integer number', () => {
    const result = sectionSchema.parse({ name: 'Toddlers', min_age_months: 12 });
    expect(result.min_age_months).toBe(12);
  });

  it('accepts zero as a valid age', () => {
    const result = sectionSchema.parse({ name: 'Toddlers', min_age_months: 0 });
    expect(result.min_age_months).toBe(0);
  });

  it('accepts a numeric string and coerces to number', () => {
    const result = sectionSchema.parse({ name: 'Toddlers', min_age_months: '24' });
    expect(result.min_age_months).toBe(24);
  });

  it('rejects negative numbers', () => {
    const result = sectionSchema.safeParse({ name: 'Toddlers', min_age_months: -1 });
    expect(result.success).toBe(false);
    if (!result.success) {
      const paths = result.error.issues.map((i) => i.path.join('.'));
      expect(paths).toContain('min_age_months');
    }
  });

  it('rejects floating-point numbers (not integers)', () => {
    const result = sectionSchema.safeParse({ name: 'Toddlers', min_age_months: 6.5 });
    expect(result.success).toBe(false);
    if (!result.success) {
      const paths = result.error.issues.map((i) => i.path.join('.'));
      expect(paths).toContain('min_age_months');
    }
  });
});

// ---------------------------------------------------------------------------
// sectionSchema – name field
// ---------------------------------------------------------------------------
describe('sectionSchema – name validation', () => {
  it('requires name to be present', () => {
    const result = sectionSchema.safeParse({});
    expect(result.success).toBe(false);
  });

  it('rejects an empty name', () => {
    const result = sectionSchema.safeParse({ name: '' });
    expect(result.success).toBe(false);
  });

  it('accepts a single-character name', () => {
    const result = sectionSchema.safeParse({ name: 'A' });
    expect(result.success).toBe(true);
  });

  it('accepts a 255-character name', () => {
    const result = sectionSchema.safeParse({ name: 'A'.repeat(255) });
    expect(result.success).toBe(true);
  });

  it('rejects a 256-character name', () => {
    const result = sectionSchema.safeParse({ name: 'A'.repeat(256) });
    expect(result.success).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// sectionSchema – age constraints optional
// ---------------------------------------------------------------------------
describe('sectionSchema – optional age fields', () => {
  it('accepts name-only without any age fields', () => {
    const result = sectionSchema.parse({ name: 'Infants' });
    expect(result).toEqual({ name: 'Infants' });
  });

  it('accepts only min_age_months without max', () => {
    const result = sectionSchema.parse({ name: 'Infants', min_age_months: 0 });
    expect(result).toEqual({ name: 'Infants', min_age_months: 0 });
  });

  it('accepts only max_age_months without min', () => {
    const result = sectionSchema.parse({ name: 'Infants', max_age_months: 36 });
    expect(result).toEqual({ name: 'Infants', max_age_months: 36 });
  });
});

// ---------------------------------------------------------------------------
// sectionSchema – min < max refinement
// ---------------------------------------------------------------------------
describe('sectionSchema – min/max age refinement', () => {
  it('accepts min_age_months < max_age_months', () => {
    const result = sectionSchema.safeParse({
      name: 'Toddlers',
      min_age_months: 12,
      max_age_months: 36,
    });
    expect(result.success).toBe(true);
  });

  it('accepts min_age_months = 0 with max_age_months = 1', () => {
    const result = sectionSchema.safeParse({
      name: 'Newborns',
      min_age_months: 0,
      max_age_months: 1,
    });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.min_age_months).toBe(0);
      expect(result.data.max_age_months).toBe(1);
    }
  });

  it('rejects equal min_age_months and max_age_months', () => {
    const result = sectionSchema.safeParse({
      name: 'Toddlers',
      min_age_months: 24,
      max_age_months: 24,
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].path).toContain('max_age_months');
      expect(result.error.issues[0].message).toBe(
        'min_age_months must be less than max_age_months'
      );
    }
  });

  it('rejects min_age_months > max_age_months', () => {
    const result = sectionSchema.safeParse({
      name: 'Toddlers',
      min_age_months: 36,
      max_age_months: 12,
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].path).toContain('max_age_months');
      expect(result.error.issues[0].message).toBe(
        'min_age_months must be less than max_age_months'
      );
    }
  });

  it('skips refinement when min is null (empty string) and max is set', () => {
    const result = sectionSchema.safeParse({
      name: 'Toddlers',
      min_age_months: '',
      max_age_months: 36,
    });
    expect(result.success).toBe(true);
  });

  it('skips refinement when max is null (empty string) and min is set', () => {
    const result = sectionSchema.safeParse({
      name: 'Toddlers',
      min_age_months: 12,
      max_age_months: '',
    });
    expect(result.success).toBe(true);
  });

  it('skips refinement when both are null', () => {
    const result = sectionSchema.safeParse({
      name: 'Toddlers',
      min_age_months: null,
      max_age_months: null,
    });
    expect(result.success).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// sectionSchema – combined validation
// ---------------------------------------------------------------------------
describe('sectionSchema – combined invalid inputs', () => {
  it('reports both name and age errors simultaneously', () => {
    const result = sectionSchema.safeParse({
      name: '',
      min_age_months: -5,
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const paths = result.error.issues.map((i) => i.path.join('.'));
      expect(paths).toContain('name');
      expect(paths).toContain('min_age_months');
    }
  });
});
