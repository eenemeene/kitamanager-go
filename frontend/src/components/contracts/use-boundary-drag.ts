'use client';

import { useState, useCallback, useEffect, useRef } from 'react';
import { differenceInCalendarDays, parseISO } from 'date-fns';
import { computeNewBoundary, type BaseContract } from './timeline-utils';
import type { ContractBatchUpdateItem } from '@/lib/api/types';
import { formatDateForApi } from '@/lib/utils/formatting';

export interface DragState {
  boundaryIndex: number;
  upperIndex: number;
  lowerIndex: number;
  startY: number;
  originalToDate: string;
  newTo: string;
  newFrom: string;
}

interface UseBoundaryDragOptions {
  contracts: BaseContract[];
  onDragEnd: (updates: ContractBatchUpdateItem[]) => void;
}

const PIXELS_PER_DAY = 2;

export function useBoundaryDrag({ contracts, onDragEnd }: UseBoundaryDragOptions) {
  const [dragState, setDragState] = useState<DragState | null>(null);
  const dragRef = useRef<DragState | null>(null);

  const handlePointerDown = useCallback(
    (boundaryIndex: number, e: React.PointerEvent) => {
      // In the timeline, contracts are sorted newest-first.
      // boundaryIndex refers to the position between items.
      // We need to find which two contracts this boundary is between.
      // The boundary at position N is between contract[N] (upper/newer) and contract[N+1] (lower/older).
      // But we pass upperIndex/lowerIndex from the timeline items.
      // For simplicity, we just receive the indices directly.

      // Find the boundary's contracts. The caller passes the boundaryIndex
      // which corresponds to the index in the sorted contracts array.
      // upperIndex = boundaryIndex, lowerIndex = boundaryIndex + 1
      const upperIndex = boundaryIndex;
      const lowerIndex = boundaryIndex + 1;
      const upper = contracts[upperIndex];
      const lower = contracts[lowerIndex];

      if (!upper || !lower || !lower.to) return;

      // The boundary date is lower.to / upper.from
      // Wait — contracts are sorted newest first. So:
      // contracts[0] is newest, contracts[1] is next, etc.
      // Adjacent means: contracts[i].from - 1 day = contracts[i-1]... no.
      // Actually: the "boundary" is between the end of the lower/older contract
      // and the start of the upper/newer contract.
      // In newest-first order: contracts[upperIndex] is newer, contracts[lowerIndex] is older.
      // The boundary is: contracts[lowerIndex].to | contracts[upperIndex].from

      const originalToDate = lower.to; // the older contract's end date

      const state: DragState = {
        boundaryIndex,
        upperIndex,
        lowerIndex,
        startY: e.clientY,
        originalToDate,
        newTo: originalToDate,
        newFrom: upper.from,
      };

      dragRef.current = state;
      setDragState(state);
      (e.target as HTMLElement).setPointerCapture(e.pointerId);
    },
    [contracts]
  );

  useEffect(() => {
    if (!dragState) return;

    const upper = contracts[dragState.upperIndex]; // newer contract
    const lower = contracts[dragState.lowerIndex]; // older contract

    // Constraints:
    // minDate for lower.to: lower.from (older contract must keep at least 1 day)
    const minDate = lower.from;
    // maxDate for upper.from: upper.to (newer contract must keep at least 1 day)
    // If upper has no `to`, use a far-future date
    const maxDate = upper.to || null;

    const handlePointerMove = (e: PointerEvent) => {
      const current = dragRef.current;
      if (!current) return;

      const deltaY = e.clientY - current.startY;
      // Moving down = positive deltaY = boundary moves later (down in timeline = older)
      // In our newest-first layout, down means older/earlier dates
      // So positive deltaY should shift boundary to earlier date (negative days)
      const deltaDays = -Math.round(deltaY / PIXELS_PER_DAY);

      const { newTo, newFrom } = computeNewBoundary(
        current.originalToDate,
        deltaDays,
        minDate,
        maxDate
      );

      const newState = { ...current, newTo, newFrom };
      dragRef.current = newState;
      setDragState(newState);
    };

    const handlePointerUp = () => {
      const current = dragRef.current;
      if (!current) return;

      // Check if the boundary actually moved
      if (current.newTo !== current.originalToDate) {
        const updates: ContractBatchUpdateItem[] = [
          { id: lower.id, to: formatDateForApi(current.newTo) },
          { id: upper.id, from: formatDateForApi(current.newFrom) ?? undefined },
        ];
        onDragEnd(updates);
      }

      dragRef.current = null;
      setDragState(null);
    };

    document.addEventListener('pointermove', handlePointerMove);
    document.addEventListener('pointerup', handlePointerUp);

    return () => {
      document.removeEventListener('pointermove', handlePointerMove);
      document.removeEventListener('pointerup', handlePointerUp);
    };
  }, [dragState, contracts, onDragEnd]);

  return { dragState, handlePointerDown };
}

/**
 * Compute a reasonable pixels-per-day scale based on the contracts' total span.
 * Not currently used (using fixed PIXELS_PER_DAY), but available for future use.
 */
export function computePixelsPerDay(contracts: BaseContract[], containerHeight: number): number {
  if (contracts.length === 0) return PIXELS_PER_DAY;

  const oldest = contracts[contracts.length - 1];
  const newest = contracts[0];
  const newestEnd = newest.to || newest.from;
  const totalDays = Math.abs(differenceInCalendarDays(parseISO(newestEnd), parseISO(oldest.from)));

  if (totalDays === 0) return PIXELS_PER_DAY;
  const scale = containerHeight / totalDays;
  return Math.max(0.5, Math.min(5, scale));
}
