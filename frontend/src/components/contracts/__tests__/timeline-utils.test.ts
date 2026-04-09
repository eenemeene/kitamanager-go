import { areAdjacent, buildTimelineItems, type BaseContract } from '../timeline-utils';

describe('areAdjacent', () => {
  it('returns true when lower.to + 1 day = upper.from', () => {
    const lower: BaseContract = {
      id: 1,
      from: '2024-01-01T00:00:00Z',
      to: '2024-06-30T00:00:00Z',
    };
    const upper: BaseContract = {
      id: 2,
      from: '2024-07-01T00:00:00Z',
      to: '2024-12-31T00:00:00Z',
    };
    expect(areAdjacent(upper, lower)).toBe(false); // areAdjacent checks upper.to+1=lower.from
    expect(areAdjacent(lower, upper)).toBe(true); // lower.to+1 = upper.from
  });

  it('returns false when there is a gap between contracts', () => {
    const lower: BaseContract = {
      id: 1,
      from: '2024-01-01T00:00:00Z',
      to: '2024-03-31T00:00:00Z',
    };
    const upper: BaseContract = {
      id: 2,
      from: '2024-07-01T00:00:00Z',
      to: '2024-12-31T00:00:00Z',
    };
    expect(areAdjacent(lower, upper)).toBe(false);
    expect(areAdjacent(upper, lower)).toBe(false);
  });

  it('returns false when upper has no to date', () => {
    const upper: BaseContract = { id: 2, from: '2024-07-01T00:00:00Z' };
    const lower: BaseContract = {
      id: 1,
      from: '2024-01-01T00:00:00Z',
      to: '2024-06-30T00:00:00Z',
    };
    expect(areAdjacent(upper, lower)).toBe(false);
  });

  it('returns false when upper.to is null', () => {
    const upper: BaseContract = { id: 2, from: '2024-07-01T00:00:00Z', to: null };
    const lower: BaseContract = {
      id: 1,
      from: '2024-01-01T00:00:00Z',
      to: '2024-06-30T00:00:00Z',
    };
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
      { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
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
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-03-31T00:00:00Z' },
    ];
    const items = buildTimelineItems(contracts);
    expect(items).toHaveLength(5);
    expect(items[1].type).toBe('boundary'); // between id:3 and id:2
    expect(items[3].type).toBe('gap'); // between id:2 and id:1
  });
});
