import {
  areAdjacent,
  buildTimelineItems,
  computeNewBoundary,
  applyDragToContracts,
  type BaseContract,
} from '../timeline-utils';

describe('areAdjacent', () => {
  it('returns true when lower.to + 1 day = upper.from', () => {
    const lower: BaseContract = { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' };
    const upper: BaseContract = { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' };
    expect(areAdjacent(upper, lower)).toBe(false); // areAdjacent checks upper.to+1=lower.from
    // But buildTimelineItems checks both directions, and the correct call for newest-first is:
    expect(areAdjacent(lower, upper)).toBe(true); // lower.to+1 = upper.from
  });

  it('returns false when there is a gap between contracts', () => {
    const lower: BaseContract = { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-03-31T00:00:00Z' };
    const upper: BaseContract = { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' };
    expect(areAdjacent(lower, upper)).toBe(false);
    expect(areAdjacent(upper, lower)).toBe(false);
  });

  it('returns false when upper has no to date', () => {
    const upper: BaseContract = { id: 2, from: '2024-07-01T00:00:00Z' };
    const lower: BaseContract = { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' };
    expect(areAdjacent(upper, lower)).toBe(false);
  });

  it('returns false when upper.to is null', () => {
    const upper: BaseContract = { id: 2, from: '2024-07-01T00:00:00Z', to: null };
    const lower: BaseContract = { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' };
    expect(areAdjacent(upper, lower)).toBe(false);
  });
});

describe('buildTimelineItems', () => {
  it('returns empty array for no contracts', () => {
    expect(buildTimelineItems([])).toEqual([]);
  });

  it('returns single segment for one contract', () => {
    const contracts: BaseContract[] = [
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
    ];
    const items = buildTimelineItems(contracts);
    expect(items).toEqual([{ type: 'segment', contract: contracts[0], index: 0 }]);
  });

  it('inserts boundary between adjacent contracts (newest-first)', () => {
    const contracts: BaseContract[] = [
      { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' }, // newer
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' }, // older
    ];
    const items = buildTimelineItems(contracts);
    expect(items).toHaveLength(3);
    expect(items[0]).toEqual({ type: 'segment', contract: contracts[0], index: 0 });
    expect(items[1]).toEqual({ type: 'boundary', upperIndex: 0, lowerIndex: 1 });
    expect(items[2]).toEqual({ type: 'segment', contract: contracts[1], index: 1 });
  });

  it('inserts gap for non-adjacent contracts', () => {
    const contracts: BaseContract[] = [
      { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-03-31T00:00:00Z' },
    ];
    const items = buildTimelineItems(contracts);
    expect(items).toHaveLength(3);
    expect(items[1].type).toBe('gap');
    if (items[1].type === 'gap') {
      expect(items[1].gapDays).toBeGreaterThan(0);
    }
  });

  it('handles three adjacent contracts', () => {
    const contracts: BaseContract[] = [
      { id: 3, from: '2025-01-01T00:00:00Z', to: '2025-06-30T00:00:00Z' },
      { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
    ];
    const items = buildTimelineItems(contracts);
    expect(items).toHaveLength(5); // 3 segments + 2 boundaries
    expect(items[0].type).toBe('segment');
    expect(items[1].type).toBe('boundary');
    expect(items[2].type).toBe('segment');
    expect(items[3].type).toBe('boundary');
    expect(items[4].type).toBe('segment');
  });

  it('handles mix of adjacent and gap', () => {
    const contracts: BaseContract[] = [
      { id: 3, from: '2025-01-01T00:00:00Z', to: '2025-06-30T00:00:00Z' },
      { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-03-31T00:00:00Z' }, // gap before id:2
    ];
    const items = buildTimelineItems(contracts);
    expect(items).toHaveLength(5);
    expect(items[1].type).toBe('boundary'); // between id:3 and id:2
    expect(items[3].type).toBe('gap'); // between id:2 and id:1
  });
});

describe('computeNewBoundary', () => {
  it('shifts forward by N days', () => {
    const result = computeNewBoundary(
      '2024-06-30T00:00:00Z',
      5,
      '2024-01-01T00:00:00Z',
      '2024-12-31T00:00:00Z'
    );
    expect(result.newTo).toBe('2024-07-05T00:00:00Z');
    expect(result.newFrom).toBe('2024-07-06T00:00:00Z');
  });

  it('shifts backward by N days', () => {
    const result = computeNewBoundary(
      '2024-06-30T00:00:00Z',
      -10,
      '2024-01-01T00:00:00Z',
      '2024-12-31T00:00:00Z'
    );
    expect(result.newTo).toBe('2024-06-20T00:00:00Z');
    expect(result.newFrom).toBe('2024-06-21T00:00:00Z');
  });

  it('clamps to min date (cannot shrink upper contract to nothing)', () => {
    const result = computeNewBoundary(
      '2024-06-30T00:00:00Z',
      -365,
      '2024-06-01T00:00:00Z',
      '2024-12-31T00:00:00Z'
    );
    expect(result.newTo).toBe('2024-06-01T00:00:00Z');
    expect(result.newFrom).toBe('2024-06-02T00:00:00Z');
  });

  it('clamps to max date (cannot shrink lower contract to nothing)', () => {
    const result = computeNewBoundary(
      '2024-06-30T00:00:00Z',
      365,
      '2024-01-01T00:00:00Z',
      '2024-12-31T00:00:00Z'
    );
    expect(result.newTo).toBe('2024-12-30T00:00:00Z');
    expect(result.newFrom).toBe('2024-12-31T00:00:00Z');
  });

  it('handles null maxDate (ongoing lower contract)', () => {
    const result = computeNewBoundary('2024-06-30T00:00:00Z', 365, '2024-01-01T00:00:00Z', null);
    expect(result.newTo).toBe('2025-06-30T00:00:00Z');
    expect(result.newFrom).toBe('2025-07-01T00:00:00Z');
  });

  it('handles zero delta', () => {
    const result = computeNewBoundary(
      '2024-06-30T00:00:00Z',
      0,
      '2024-01-01T00:00:00Z',
      '2024-12-31T00:00:00Z'
    );
    expect(result.newTo).toBe('2024-06-30T00:00:00Z');
    expect(result.newFrom).toBe('2024-07-01T00:00:00Z');
  });
});

describe('applyDragToContracts', () => {
  it('applies newTo to upper contract and newFrom to lower contract', () => {
    const contracts: BaseContract[] = [
      { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
    ];
    const result = applyDragToContracts(
      contracts,
      0,
      1,
      '2024-08-01T00:00:00Z',
      '2024-08-02T00:00:00Z'
    );
    expect(result[0].to).toBe('2024-08-01T00:00:00Z');
    expect(result[1].from).toBe('2024-08-02T00:00:00Z');
    // Other fields unchanged
    expect(result[0].from).toBe('2024-07-01T00:00:00Z');
    expect(result[1].to).toBe('2024-06-30T00:00:00Z');
  });

  it('does not modify contracts at other indices', () => {
    const contracts: BaseContract[] = [
      { id: 3, from: '2025-01-01T00:00:00Z', to: '2025-06-30T00:00:00Z' },
      { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
    ];
    const result = applyDragToContracts(
      contracts,
      1,
      2,
      '2024-08-01T00:00:00Z',
      '2024-08-02T00:00:00Z'
    );
    expect(result[0]).toBe(contracts[0]); // same reference, untouched
    expect(result[1].to).toBe('2024-08-01T00:00:00Z');
    expect(result[2].from).toBe('2024-08-02T00:00:00Z');
  });

  it('returns a new array (does not mutate original)', () => {
    const contracts: BaseContract[] = [
      { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
    ];
    const result = applyDragToContracts(
      contracts,
      0,
      1,
      '2024-08-01T00:00:00Z',
      '2024-08-02T00:00:00Z'
    );
    expect(result).not.toBe(contracts);
    expect(contracts[0].to).toBe('2024-12-31T00:00:00Z'); // original unchanged
  });
});
