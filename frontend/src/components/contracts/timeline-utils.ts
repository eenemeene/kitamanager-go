import { parseISO, addDays, isSameDay, differenceInCalendarDays } from 'date-fns';

export interface BaseContract {
  id: number;
  from: string;
  to?: string | null;
}

export type TimelineItem =
  | { type: 'segment'; contract: BaseContract; index: number }
  | { type: 'boundary'; upperIndex: number; lowerIndex: number }
  | { type: 'gap'; upperIndex: number; lowerIndex: number; gapDays: number };

/**
 * Check if two contracts are adjacent (A.to + 1 day = B.from).
 * Contracts must be ordered: `upper` is newer (ends later), `lower` is older (starts earlier).
 * In the timeline, contracts are sorted newest-first, so `upper` appears first in the array.
 */
export function areAdjacent(upper: BaseContract, lower: BaseContract): boolean {
  if (!upper.to) return false;
  const upperEnd = parseISO(upper.to);
  const lowerStart = parseISO(lower.from);
  return isSameDay(addDays(upperEnd, 1), lowerStart);
}

/**
 * Build interleaved timeline items from contracts sorted newest-first.
 * Inserts boundary handles between adjacent contracts and gap indicators otherwise.
 */
export function buildTimelineItems(contracts: BaseContract[]): TimelineItem[] {
  if (contracts.length === 0) return [];

  const items: TimelineItem[] = [{ type: 'segment', contract: contracts[0], index: 0 }];

  for (let i = 1; i < contracts.length; i++) {
    const upper = contracts[i - 1]; // newer contract (appears above in timeline)
    const lower = contracts[i]; // older contract (appears below)

    if (areAdjacent(lower, upper)) {
      // lower.to + 1 = upper.from → they are adjacent
      items.push({ type: 'boundary', upperIndex: i - 1, lowerIndex: i });
    } else if (areAdjacent(upper, lower)) {
      // upper.to + 1 = lower.from → also adjacent (different order)
      items.push({ type: 'boundary', upperIndex: i - 1, lowerIndex: i });
    } else {
      // Gap between contracts
      const gapDays = computeGapDays(upper, lower);
      items.push({ type: 'gap', upperIndex: i - 1, lowerIndex: i, gapDays });
    }

    items.push({ type: 'segment', contract: lower, index: i });
  }

  return items;
}

function computeGapDays(upper: BaseContract, lower: BaseContract): number {
  // upper is newer (later dates), lower is older
  // Gap is between lower.to and upper.from
  if (lower.to) {
    return Math.abs(differenceInCalendarDays(parseISO(upper.from), parseISO(lower.to)));
  }
  return 0;
}
