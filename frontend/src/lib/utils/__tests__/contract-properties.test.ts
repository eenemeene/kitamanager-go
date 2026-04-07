import {
  propertiesToValues,
  propertiesToLabelKeys,
  getPropertyValue,
  getScalarPropertyValue,
  setProperty,
  removePropertyByValue,
  hasPropertyValue,
  getKeyForValue,
} from '../contract-properties';

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

  it('flattens scalar string values', () => {
    const result = propertiesToValues({ care_type: 'ganztag', ndh: 'yes' });
    expect(result).toContain('ganztag');
    expect(result).toContain('yes');
    expect(result).toHaveLength(2);
  });

  it('flattens array values', () => {
    const result = propertiesToValues({
      supplements: ['ndh', 'mss'],
    });
    expect(result).toEqual(['ndh', 'mss']);
  });

  it('handles mix of scalar and array values', () => {
    const result = propertiesToValues({
      care_type: 'ganztag',
      supplements: ['ndh', 'mss'],
    });
    expect(result).toContain('ganztag');
    expect(result).toContain('ndh');
    expect(result).toContain('mss');
    expect(result).toHaveLength(3);
  });

  it('handles empty array value', () => {
    const result = propertiesToValues({ supplements: [] });
    expect(result).toEqual([]);
  });
});

// ---------------------------------------------------------------------------
// getPropertyValue
// ---------------------------------------------------------------------------
describe('getPropertyValue', () => {
  it('returns undefined for undefined properties', () => {
    expect(getPropertyValue(undefined, 'care_type')).toBeUndefined();
  });

  it('returns undefined for missing key', () => {
    expect(getPropertyValue({ care_type: 'ganztag' }, 'missing')).toBeUndefined();
  });

  it('returns scalar string value', () => {
    expect(getPropertyValue({ care_type: 'ganztag' }, 'care_type')).toBe('ganztag');
  });

  it('returns array value', () => {
    expect(getPropertyValue({ supplements: ['ndh', 'mss'] }, 'supplements')).toEqual([
      'ndh',
      'mss',
    ]);
  });
});

// ---------------------------------------------------------------------------
// getScalarPropertyValue
// ---------------------------------------------------------------------------
describe('getScalarPropertyValue', () => {
  it('returns undefined for undefined properties', () => {
    expect(getScalarPropertyValue(undefined, 'care_type')).toBeUndefined();
  });

  it('returns undefined for missing key', () => {
    expect(getScalarPropertyValue({ care_type: 'ganztag' }, 'missing')).toBeUndefined();
  });

  it('returns string value', () => {
    expect(getScalarPropertyValue({ care_type: 'ganztag' }, 'care_type')).toBe('ganztag');
  });

  it('returns undefined for array value', () => {
    expect(getScalarPropertyValue({ supplements: ['ndh'] }, 'supplements')).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// setProperty
// ---------------------------------------------------------------------------
describe('setProperty', () => {
  it('creates new properties from undefined', () => {
    const result = setProperty(undefined, 'care_type', 'ganztag');
    expect(result).toEqual({ care_type: 'ganztag' });
  });

  it('adds to existing properties', () => {
    const result = setProperty({ care_type: 'ganztag' }, 'ndh', 'yes');
    expect(result).toEqual({ care_type: 'ganztag', ndh: 'yes' });
  });

  it('replaces existing key', () => {
    const result = setProperty({ care_type: 'ganztag' }, 'care_type', 'halbtag');
    expect(result).toEqual({ care_type: 'halbtag' });
  });

  it('sets array value', () => {
    const result = setProperty(undefined, 'supplements', ['ndh', 'mss']);
    expect(result).toEqual({ supplements: ['ndh', 'mss'] });
  });

  it('does not mutate original', () => {
    const original = { care_type: 'ganztag' };
    setProperty(original, 'ndh', 'yes');
    expect(original).toEqual({ care_type: 'ganztag' });
  });
});

// ---------------------------------------------------------------------------
// removePropertyByValue
// ---------------------------------------------------------------------------
describe('removePropertyByValue', () => {
  it('returns undefined for undefined properties', () => {
    expect(removePropertyByValue(undefined, 'ganztag')).toBeUndefined();
  });

  it('removes matching scalar value', () => {
    const result = removePropertyByValue({ care_type: 'ganztag', ndh: 'yes' }, 'ganztag');
    expect(result).toEqual({ ndh: 'yes' });
  });

  it('returns undefined when removing last property', () => {
    const result = removePropertyByValue({ care_type: 'ganztag' }, 'ganztag');
    expect(result).toBeUndefined();
  });

  it('returns all properties when value not found', () => {
    const props = { care_type: 'ganztag' };
    const result = removePropertyByValue(props, 'nonexistent');
    expect(result).toEqual({ care_type: 'ganztag' });
  });

  it('does not mutate original', () => {
    const original = { care_type: 'ganztag', ndh: 'yes' };
    removePropertyByValue(original, 'ganztag');
    expect(original).toEqual({ care_type: 'ganztag', ndh: 'yes' });
  });
});

// ---------------------------------------------------------------------------
// hasPropertyValue
// ---------------------------------------------------------------------------
describe('hasPropertyValue', () => {
  it('returns false for undefined properties', () => {
    expect(hasPropertyValue(undefined, 'ganztag')).toBe(false);
  });

  it('returns true for existing scalar value', () => {
    expect(hasPropertyValue({ care_type: 'ganztag' }, 'ganztag')).toBe(true);
  });

  it('returns false for non-existing value', () => {
    expect(hasPropertyValue({ care_type: 'ganztag' }, 'halbtag')).toBe(false);
  });

  it('returns false for empty properties', () => {
    expect(hasPropertyValue({}, 'ganztag')).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// getKeyForValue
// ---------------------------------------------------------------------------
describe('getKeyForValue', () => {
  it('returns undefined for undefined properties', () => {
    expect(getKeyForValue(undefined, 'ganztag')).toBeUndefined();
  });

  it('returns key for matching scalar value', () => {
    expect(getKeyForValue({ care_type: 'ganztag' }, 'ganztag')).toBe('care_type');
  });

  it('returns undefined for non-existing value', () => {
    expect(getKeyForValue({ care_type: 'ganztag' }, 'halbtag')).toBeUndefined();
  });

  it('returns first matching key when multiple keys exist', () => {
    const result = getKeyForValue({ care_type: 'ganztag', ndh: 'yes' }, 'yes');
    expect(result).toBe('ndh');
  });

  it('returns undefined for empty properties', () => {
    expect(getKeyForValue({}, 'ganztag')).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// propertiesToLabelKeys
// ---------------------------------------------------------------------------
describe('propertiesToLabelKeys', () => {
  it('returns empty array for undefined', () => {
    expect(propertiesToLabelKeys(undefined)).toEqual([]);
  });

  it('returns empty array for empty object', () => {
    expect(propertiesToLabelKeys({})).toEqual([]);
  });

  it('creates key--value pairs for scalar values', () => {
    const result = propertiesToLabelKeys({ care_type: 'ganztag', ndh: 'ndh' });
    expect(result).toEqual(['care_type--ganztag', 'ndh--ndh']);
  });

  it('creates key--value pairs for array values', () => {
    const result = propertiesToLabelKeys({ supplements: ['ndh', 'mss'] });
    expect(result).toEqual(['supplements--ndh', 'supplements--mss']);
  });

  it('handles mix of scalar and array values', () => {
    const result = propertiesToLabelKeys({
      care_type: 'ganztag',
      supplements: ['ndh', 'mss'],
    });
    expect(result).toContain('care_type--ganztag');
    expect(result).toContain('supplements--ndh');
    expect(result).toContain('supplements--mss');
    expect(result).toHaveLength(3);
  });

  it('handles empty array value', () => {
    const result = propertiesToLabelKeys({ supplements: [] });
    expect(result).toEqual([]);
  });
});
