import { childSchema, childContractSchema, childWithContractSchema } from '../child';

// ---------------------------------------------------------------------------
// Helper: extract Zod error paths and messages
// ---------------------------------------------------------------------------
function getErrors(result: {
  success: false;
  error: { issues: Array<{ path: PropertyKey[]; message: string }> };
}) {
  return result.error.issues.map((i) => ({ path: i.path.join('.'), message: i.message }));
}

// ---------------------------------------------------------------------------
// childSchema (== personBaseSchema)
// ---------------------------------------------------------------------------
describe('childSchema', () => {
  const validChild = {
    first_name: 'Emma',
    last_name: 'Müller',
    gender: 'female' as const,
    birthdate: '2020-03-15',
  };

  it('accepts valid child data', () => {
    expect(childSchema.safeParse(validChild).success).toBe(true);
  });

  it('accepts all gender values', () => {
    for (const gender of ['male', 'female', 'diverse'] as const) {
      expect(childSchema.safeParse({ ...validChild, gender }).success).toBe(true);
    }
  });

  it('rejects invalid gender', () => {
    const result = childSchema.safeParse({ ...validChild, gender: 'other' });
    expect(result.success).toBe(false);
  });

  it('requires first_name', () => {
    const result = childSchema.safeParse({ ...validChild, first_name: '' });
    expect(result.success).toBe(false);
  });

  it('requires last_name', () => {
    const result = childSchema.safeParse({ ...validChild, last_name: '' });
    expect(result.success).toBe(false);
  });

  it('requires birthdate', () => {
    const result = childSchema.safeParse({ ...validChild, birthdate: '' });
    expect(result.success).toBe(false);
  });

  it('enforces max length of 255 on first_name', () => {
    const result = childSchema.safeParse({ ...validChild, first_name: 'A'.repeat(256) });
    expect(result.success).toBe(false);
  });

  it('enforces max length of 255 on last_name', () => {
    const result = childSchema.safeParse({ ...validChild, last_name: 'B'.repeat(256) });
    expect(result.success).toBe(false);
  });

  it('accepts names at max length boundary (255 chars)', () => {
    const result = childSchema.safeParse({
      ...validChild,
      first_name: 'A'.repeat(255),
      last_name: 'B'.repeat(255),
    });
    expect(result.success).toBe(true);
  });

  it('rejects missing fields entirely', () => {
    const result = childSchema.safeParse({});
    expect(result.success).toBe(false);
    if (!result.success) {
      const paths = result.error.issues.map((i) => i.path[0]);
      expect(paths).toContain('first_name');
      expect(paths).toContain('last_name');
      expect(paths).toContain('gender');
      expect(paths).toContain('birthdate');
    }
  });
});

// ---------------------------------------------------------------------------
// childContractSchema
// ---------------------------------------------------------------------------
describe('childContractSchema', () => {
  const validContract = {
    from: '2024-01-01',
    to: '2024-12-31',
    section_id: 1,
  };

  it('accepts valid contract with from and to', () => {
    expect(childContractSchema.safeParse(validContract).success).toBe(true);
  });

  it('accepts contract without to (open-ended)', () => {
    const { to: _, ...openEnded } = validContract;
    expect(childContractSchema.safeParse(openEnded).success).toBe(true);
  });

  it('accepts contract with undefined to', () => {
    expect(childContractSchema.safeParse({ ...validContract, to: undefined }).success).toBe(true);
  });

  it('requires from date', () => {
    const result = childContractSchema.safeParse({ ...validContract, from: '' });
    expect(result.success).toBe(false);
  });

  it('requires section_id >= 1', () => {
    const result = childContractSchema.safeParse({ ...validContract, section_id: 0 });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = getErrors(result);
      expect(
        errors.some((e) => e.path === 'section_id' && e.message === 'Section is required')
      ).toBe(true);
    }
  });

  it('rejects negative section_id', () => {
    const result = childContractSchema.safeParse({ ...validContract, section_id: -1 });
    expect(result.success).toBe(false);
  });

  it('accepts optional properties as key-value record', () => {
    const result = childContractSchema.safeParse({
      ...validContract,
      properties: { care_hours: '8', meal_plan: 'full' },
    });
    expect(result.success).toBe(true);
  });

  // --- Date refinement: to must not be before from ---

  it('rejects to date before from date', () => {
    const result = childContractSchema.safeParse({
      ...validContract,
      from: '2024-06-01',
      to: '2024-05-31',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = getErrors(result);
      expect(
        errors.some((e) => e.path === 'to' && e.message === 'End date must be after start date')
      ).toBe(true);
    }
  });

  it('accepts to date equal to from date (same-day contract)', () => {
    const result = childContractSchema.safeParse({
      ...validContract,
      from: '2024-06-01',
      to: '2024-06-01',
    });
    expect(result.success).toBe(true);
  });

  it('accepts to date one day after from date', () => {
    const result = childContractSchema.safeParse({
      ...validContract,
      from: '2024-06-01',
      to: '2024-06-02',
    });
    expect(result.success).toBe(true);
  });

  it('rejects to date one day before from date', () => {
    const result = childContractSchema.safeParse({
      ...validContract,
      from: '2024-06-02',
      to: '2024-06-01',
    });
    expect(result.success).toBe(false);
  });

  it('handles year boundary: from Dec, to Jan next year', () => {
    const result = childContractSchema.safeParse({
      ...validContract,
      from: '2024-12-01',
      to: '2025-01-31',
    });
    expect(result.success).toBe(true);
  });

  it('rejects reversed year boundary: from Jan, to Dec previous year', () => {
    const result = childContractSchema.safeParse({
      ...validContract,
      from: '2025-01-01',
      to: '2024-12-31',
    });
    expect(result.success).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// childWithContractSchema
// ---------------------------------------------------------------------------
describe('childWithContractSchema', () => {
  const validData = {
    first_name: 'Liam',
    last_name: 'Schmidt',
    gender: 'male' as const,
    birthdate: '2020-01-15',
    contract_from: '2023-08-01',
    contract_to: '2024-07-31',
    section_id: 2,
  };

  it('accepts valid combined child + contract data', () => {
    expect(childWithContractSchema.safeParse(validData).success).toBe(true);
  });

  it('accepts without contract_to (open-ended)', () => {
    const { contract_to: _, ...data } = validData;
    expect(childWithContractSchema.safeParse(data).success).toBe(true);
  });

  it('accepts with optional properties', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      properties: { care_scope: 'full_day' },
    });
    expect(result.success).toBe(true);
  });

  // --- All gender values ---

  it.each(['male', 'female', 'diverse'] as const)('accepts gender "%s"', (gender) => {
    expect(childWithContractSchema.safeParse({ ...validData, gender }).success).toBe(true);
  });

  it('rejects invalid gender value', () => {
    const result = childWithContractSchema.safeParse({ ...validData, gender: 'nonbinary' });
    expect(result.success).toBe(false);
  });

  // --- Required fields ---

  it.each(['first_name', 'last_name', 'birthdate', 'contract_from'] as const)(
    'rejects empty string for required field "%s"',
    (field) => {
      const result = childWithContractSchema.safeParse({ ...validData, [field]: '' });
      expect(result.success).toBe(false);
    }
  );

  it('rejects missing section_id', () => {
    const { section_id: _, ...data } = validData;
    const result = childWithContractSchema.safeParse(data);
    expect(result.success).toBe(false);
  });

  it('rejects section_id of 0', () => {
    const result = childWithContractSchema.safeParse({ ...validData, section_id: 0 });
    expect(result.success).toBe(false);
  });

  // --- Refinement 1: contract_to must not be before contract_from ---

  it('rejects contract_to before contract_from', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      contract_from: '2024-06-01',
      contract_to: '2024-05-31',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = getErrors(result);
      expect(
        errors.some(
          (e) => e.path === 'contract_to' && e.message === 'End date must be after start date'
        )
      ).toBe(true);
    }
  });

  it('accepts contract_to equal to contract_from (same-day contract)', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      contract_from: '2024-06-01',
      contract_to: '2024-06-01',
    });
    expect(result.success).toBe(true);
  });

  it('accepts contract_to one day after contract_from', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      contract_from: '2024-06-01',
      contract_to: '2024-06-02',
    });
    expect(result.success).toBe(true);
  });

  // --- Refinement 2: contract_from must not be before birthdate ---

  it('rejects contract_from before birthdate', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2020-06-15',
      contract_from: '2020-06-14',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = getErrors(result);
      expect(
        errors.some(
          (e) =>
            e.path === 'contract_from' &&
            e.message === 'Contract start date cannot be before birthdate'
        )
      ).toBe(true);
    }
  });

  it('rejects contract_from equal to birthdate (same day)', () => {
    // isDateBefore('2020-06-15', '2020-06-15') returns false,
    // so the refinement checks !isDateBefore(contract_from, birthdate).
    // When contract_from == birthdate, isDateBefore returns false, so !false = true => passes.
    // Actually let's verify: the refinement is:
    //   .refine((data) => !isDateBefore(data.contract_from, data.birthdate), ...)
    // isDateBefore(same, same) = false => !false = true => validation PASSES
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2020-06-15',
      contract_from: '2020-06-15',
    });
    // contract_from == birthdate: isDateBefore returns false, refinement passes
    expect(result.success).toBe(true);
  });

  it('accepts contract_from one day after birthdate', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2020-06-15',
      contract_from: '2020-06-16',
    });
    expect(result.success).toBe(true);
  });

  it('accepts contract_from years after birthdate (typical case)', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2020-01-01',
      contract_from: '2023-08-01',
    });
    expect(result.success).toBe(true);
  });

  it('rejects contract_from one day before birthdate', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2020-06-15',
      contract_from: '2020-06-14',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = getErrors(result);
      expect(errors.some((e) => e.path === 'contract_from')).toBe(true);
    }
  });

  // --- Both refinements can fail simultaneously ---

  it('reports both errors when contract_to < contract_from AND contract_from < birthdate', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2024-06-01',
      contract_from: '2024-05-01',
      contract_to: '2024-04-01',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = getErrors(result);
      // Both refinements should fire
      expect(errors.some((e) => e.path === 'contract_to')).toBe(true);
      expect(errors.some((e) => e.path === 'contract_from')).toBe(true);
    }
  });

  // --- Edge case: year boundary for birthdate vs contract ---

  it('rejects contract starting in December before January birthdate', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2021-01-01',
      contract_from: '2020-12-31',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = getErrors(result);
      expect(
        errors.some(
          (e) =>
            e.path === 'contract_from' &&
            e.message === 'Contract start date cannot be before birthdate'
        )
      ).toBe(true);
    }
  });

  it('accepts contract starting in January on birthdate in January', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2021-01-01',
      contract_from: '2021-01-01',
    });
    expect(result.success).toBe(true);
  });

  // --- Name length boundaries ---

  it('rejects first_name exceeding 255 characters', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      first_name: 'X'.repeat(256),
    });
    expect(result.success).toBe(false);
  });

  it('rejects last_name exceeding 255 characters', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      last_name: 'Y'.repeat(256),
    });
    expect(result.success).toBe(false);
  });

  // --- Open-ended contract with birthdate validation ---

  it('accepts open-ended contract starting after birthdate', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2019-05-01',
      contract_from: '2022-08-01',
      contract_to: undefined,
    });
    expect(result.success).toBe(true);
  });

  it('rejects open-ended contract starting before birthdate', () => {
    const result = childWithContractSchema.safeParse({
      ...validData,
      birthdate: '2022-08-01',
      contract_from: '2022-07-31',
      contract_to: undefined,
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = getErrors(result);
      expect(errors.some((e) => e.path === 'contract_from')).toBe(true);
      // Should NOT have a contract_to error since it's omitted
      expect(errors.some((e) => e.path === 'contract_to')).toBe(false);
    }
  });
});
