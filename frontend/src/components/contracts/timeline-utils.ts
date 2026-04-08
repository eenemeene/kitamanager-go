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

/**
 * Compute a new boundary date given a pixel delta and scale.
 * Moving down (positive deltaY) moves the boundary later (forward in time)
 * since the timeline goes newest-at-top.
 *
 * Actually: in our timeline, newest is at top. Moving the handle DOWN means
 * making the upper contract longer (later end date) and the lower contract shorter.
 * So positive deltaY = boundary moves later.
 *
 * @param originalToDate - The upper contract's current `to` date (ISO string)
 * @param deltaDays - Number of days to shift (positive = later, negative = earlier)
 * @param minDate - Earliest allowed `to` date (upper contract's `from`)
 * @param maxDate - Latest allowed new `from` date for lower contract (lower contract's `to`, or far future)
 * @returns New `to` for upper and new `from` for lower
 */
export function computeNewBoundary(
  originalToDate: string,
  deltaDays: number,
  minDate: string,
  maxDate: string | null
): { newTo: string; newFrom: string } {
  const originalTo = parseISO(originalToDate);
  let newTo = addDays(originalTo, deltaDays);
  let newFrom = addDays(newTo, 1);

  // Clamp: newTo can't be before minDate (upper contract must have at least 1 day)
  const min = parseISO(minDate);
  if (newTo < min) {
    newTo = min;
    newFrom = addDays(newTo, 1);
  }

  // Clamp: newFrom can't be after maxDate (lower contract must have at least 1 day)
  if (maxDate) {
    const max = parseISO(maxDate);
    if (newFrom > max) {
      newFrom = max;
      newTo = addDays(newFrom, -1);
    }
  }

  return {
    newTo: formatISODate(newTo),
    newFrom: formatISODate(newFrom),
  };
}

function formatISODate(date: Date): string {
  const y = date.getFullYear();
  const m = (date.getMonth() + 1).toString().padStart(2, '0');
  const d = date.getDate().toString().padStart(2, '0');
  return `${y}-${m}-${d}T00:00:00Z`;
}

/**
 * Apply drag state to a contracts array, returning a new array with modified dates.
 * Used for optimistic visual updates during drag.
 */
export function applyDragToContracts<T extends BaseContract>(
  contracts: T[],
  upperIndex: number,
  lowerIndex: number,
  newTo: string,
  newFrom: string
): T[] {
  return contracts.map((c, i) => {
    if (i === upperIndex) return { ...c, to: newTo };
    if (i === lowerIndex) return { ...c, from: newFrom };
    return c;
  });
}
