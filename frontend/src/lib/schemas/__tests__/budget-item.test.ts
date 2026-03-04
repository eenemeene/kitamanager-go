import { budgetItemSchema, budgetItemEntrySchema, budgetItemWithEntrySchema } from '../budget-item';

// ---------------------------------------------------------------------------
// Helper: extract Zod error paths and messages
// ---------------------------------------------------------------------------
function getErrors(result: {
  success: boolean;
  error?: { issues: { path: PropertyKey[]; message: string }[] };
}) {
  if (result.success) return [];
  return result.error!.issues.map((i) => ({ path: i.path.join('.'), message: i.message }));
}

// ---------------------------------------------------------------------------
// budgetItemSchema
// ---------------------------------------------------------------------------
describe('budgetItemSchema', () => {
  it('accepts valid income item', () => {
    const result = budgetItemSchema.safeParse({
      name: 'Rent',
      category: 'income',
      per_child: false,
    });
    expect(result.success).toBe(true);
  });

  it('accepts valid expense item', () => {
    const result = budgetItemSchema.safeParse({
      name: 'Supplies',
      category: 'expense',
      per_child: true,
    });
    expect(result.success).toBe(true);
  });

  it('rejects empty name', () => {
    const result = budgetItemSchema.safeParse({ name: '', category: 'income', per_child: false });
    expect(result.success).toBe(false);
  });

  it('rejects missing name', () => {
    const result = budgetItemSchema.safeParse({ category: 'income', per_child: false });
    expect(result.success).toBe(false);
  });

  it('rejects name longer than 255 characters', () => {
    const result = budgetItemSchema.safeParse({
      name: 'x'.repeat(256),
      category: 'income',
      per_child: false,
    });
    expect(result.success).toBe(false);
  });

  it('accepts name at exactly 255 characters', () => {
    const result = budgetItemSchema.safeParse({
      name: 'x'.repeat(255),
      category: 'income',
      per_child: false,
    });
    expect(result.success).toBe(true);
  });

  it('rejects invalid category value', () => {
    const result = budgetItemSchema.safeParse({
      name: 'Rent',
      category: 'other',
      per_child: false,
    });
    expect(result.success).toBe(false);
    const errors = getErrors(result);
    expect(errors.some((e) => e.path === 'category')).toBe(true);
  });

  it('rejects missing category', () => {
    const result = budgetItemSchema.safeParse({ name: 'Rent', per_child: false });
    expect(result.success).toBe(false);
  });

  it('rejects missing per_child', () => {
    const result = budgetItemSchema.safeParse({ name: 'Rent', category: 'income' });
    expect(result.success).toBe(false);
  });

  it('rejects non-boolean per_child', () => {
    const result = budgetItemSchema.safeParse({
      name: 'Rent',
      category: 'income',
      per_child: 'yes',
    });
    expect(result.success).toBe(false);
  });

  it('rejects completely empty object', () => {
    const result = budgetItemSchema.safeParse({});
    expect(result.success).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// budgetItemEntrySchema
// ---------------------------------------------------------------------------
describe('budgetItemEntrySchema', () => {
  const validEntry = { from: '2025-01-01', amount_euros: 100 };

  it('accepts valid entry without optional fields', () => {
    const result = budgetItemEntrySchema.safeParse(validEntry);
    expect(result.success).toBe(true);
  });

  it('accepts valid entry with all fields', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-01-01',
      to: '2025-12-31',
      amount_euros: 150.5,
      notes: 'Monthly budget',
    });
    expect(result.success).toBe(true);
  });

  it('accepts zero amount', () => {
    const result = budgetItemEntrySchema.safeParse({ from: '2025-01-01', amount_euros: 0 });
    expect(result.success).toBe(true);
  });

  it('rejects negative amount', () => {
    const result = budgetItemEntrySchema.safeParse({ from: '2025-01-01', amount_euros: -1 });
    expect(result.success).toBe(false);
    const errors = getErrors(result);
    expect(errors.some((e) => e.path === 'amount_euros')).toBe(true);
  });

  it('rejects missing from date', () => {
    const result = budgetItemEntrySchema.safeParse({ amount_euros: 100 });
    expect(result.success).toBe(false);
  });

  it('rejects empty from date', () => {
    const result = budgetItemEntrySchema.safeParse({ from: '', amount_euros: 100 });
    expect(result.success).toBe(false);
  });

  it('rejects missing amount_euros', () => {
    const result = budgetItemEntrySchema.safeParse({ from: '2025-01-01' });
    expect(result.success).toBe(false);
  });

  // --- Date validation refinement ---

  it('allows to date after from date', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-01-01',
      to: '2025-06-30',
      amount_euros: 100,
    });
    expect(result.success).toBe(true);
  });

  it('allows to date equal to from date (same-day entry)', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-06-15',
      to: '2025-06-15',
      amount_euros: 100,
    });
    expect(result.success).toBe(true);
  });

  it('rejects to date before from date', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-06-15',
      to: '2025-01-01',
      amount_euros: 100,
    });
    expect(result.success).toBe(false);
    const errors = getErrors(result);
    expect(
      errors.some((e) => e.path === 'to' && e.message === 'End date must be after start date')
    ).toBe(true);
  });

  it('rejects to date one day before from date', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-06-15',
      to: '2025-06-14',
      amount_euros: 100,
    });
    expect(result.success).toBe(false);
    const errors = getErrors(result);
    expect(errors.some((e) => e.path === 'to')).toBe(true);
  });

  it('allows omitted to date (open-ended entry)', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-01-01',
      amount_euros: 200,
    });
    expect(result.success).toBe(true);
  });

  it('allows undefined to date explicitly', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-01-01',
      to: undefined,
      amount_euros: 200,
    });
    expect(result.success).toBe(true);
  });

  it('skips date refinement when to is empty string (caught by other validation)', () => {
    // Empty string is falsy, so the refinement short-circuits with !data.to
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-06-15',
      to: '',
      amount_euros: 100,
    });
    // Empty string passes the refinement (falsy), but is still a valid parse
    expect(result.success).toBe(true);
  });

  // --- Notes validation ---

  it('accepts notes at exactly 500 characters', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-01-01',
      amount_euros: 100,
      notes: 'n'.repeat(500),
    });
    expect(result.success).toBe(true);
  });

  it('rejects notes longer than 500 characters', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-01-01',
      amount_euros: 100,
      notes: 'n'.repeat(501),
    });
    expect(result.success).toBe(false);
    const errors = getErrors(result);
    expect(errors.some((e) => e.path === 'notes')).toBe(true);
  });

  it('accepts omitted notes', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-01-01',
      amount_euros: 100,
    });
    expect(result.success).toBe(true);
  });

  // --- Cross-year date boundaries ---

  it('allows to date in the next year', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2025-12-01',
      to: '2026-01-31',
      amount_euros: 100,
    });
    expect(result.success).toBe(true);
  });

  it('rejects to date in a previous year', () => {
    const result = budgetItemEntrySchema.safeParse({
      from: '2026-01-01',
      to: '2025-12-31',
      amount_euros: 100,
    });
    expect(result.success).toBe(false);
    const errors = getErrors(result);
    expect(errors.some((e) => e.path === 'to')).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// budgetItemWithEntrySchema
// ---------------------------------------------------------------------------
describe('budgetItemWithEntrySchema', () => {
  const validData = {
    name: 'Office Supplies',
    category: 'expense' as const,
    per_child: false,
    entry_from: '2025-01-01',
    entry_amount_euros: 250,
  };

  it('accepts valid combined data without optional fields', () => {
    const result = budgetItemWithEntrySchema.safeParse(validData);
    expect(result.success).toBe(true);
  });

  it('accepts valid combined data with all fields', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_to: '2025-12-31',
      entry_notes: 'Annual budget allocation',
    });
    expect(result.success).toBe(true);
  });

  // --- Budget item field validation propagates ---

  it('rejects empty name in combined schema', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, name: '' });
    expect(result.success).toBe(false);
  });

  it('rejects invalid category in combined schema', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, category: 'revenue' });
    expect(result.success).toBe(false);
  });

  it('rejects missing per_child in combined schema', () => {
    const { per_child: _, ...withoutPerChild } = validData;
    const result = budgetItemWithEntrySchema.safeParse(withoutPerChild);
    expect(result.success).toBe(false);
  });

  // --- Entry field validation ---

  it('rejects missing entry_from', () => {
    const { entry_from: _, ...withoutFrom } = validData;
    const result = budgetItemWithEntrySchema.safeParse(withoutFrom);
    expect(result.success).toBe(false);
  });

  it('rejects empty entry_from', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, entry_from: '' });
    expect(result.success).toBe(false);
  });

  it('rejects negative entry_amount_euros', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, entry_amount_euros: -5 });
    expect(result.success).toBe(false);
  });

  it('accepts zero entry_amount_euros', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, entry_amount_euros: 0 });
    expect(result.success).toBe(true);
  });

  // --- Date refinement on combined schema ---

  it('allows entry_to after entry_from', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_to: '2025-06-30',
    });
    expect(result.success).toBe(true);
  });

  it('allows entry_to equal to entry_from', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_from: '2025-03-15',
      entry_to: '2025-03-15',
    });
    expect(result.success).toBe(true);
  });

  it('rejects entry_to before entry_from', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_from: '2025-06-15',
      entry_to: '2025-01-01',
    });
    expect(result.success).toBe(false);
    const errors = getErrors(result);
    expect(
      errors.some((e) => e.path === 'entry_to' && e.message === 'End date must be after start date')
    ).toBe(true);
  });

  it('rejects entry_to one day before entry_from', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_from: '2025-06-15',
      entry_to: '2025-06-14',
    });
    expect(result.success).toBe(false);
    const errors = getErrors(result);
    expect(errors.some((e) => e.path === 'entry_to')).toBe(true);
  });

  it('allows omitted entry_to (open-ended)', () => {
    const result = budgetItemWithEntrySchema.safeParse(validData);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.entry_to).toBeUndefined();
    }
  });

  // --- Notes validation in combined schema ---

  it('rejects entry_notes longer than 500 characters', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_notes: 'x'.repeat(501),
    });
    expect(result.success).toBe(false);
  });

  it('accepts entry_notes at 500 characters', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_notes: 'x'.repeat(500),
    });
    expect(result.success).toBe(true);
  });

  // --- Cross-year date boundaries ---

  it('allows entry_to crossing year boundary', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_from: '2025-11-01',
      entry_to: '2026-02-28',
    });
    expect(result.success).toBe(true);
  });

  it('rejects entry_to in previous year relative to entry_from', () => {
    const result = budgetItemWithEntrySchema.safeParse({
      ...validData,
      entry_from: '2026-01-01',
      entry_to: '2025-12-31',
    });
    expect(result.success).toBe(false);
  });

  // --- Both category values work in combined schema ---

  it('accepts income category in combined schema', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, category: 'income' });
    expect(result.success).toBe(true);
  });

  it('accepts expense category in combined schema', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, category: 'expense' });
    expect(result.success).toBe(true);
  });

  // --- per_child boolean values ---

  it('accepts per_child true', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, per_child: true });
    expect(result.success).toBe(true);
  });

  it('accepts per_child false', () => {
    const result = budgetItemWithEntrySchema.safeParse({ ...validData, per_child: false });
    expect(result.success).toBe(true);
  });
});
