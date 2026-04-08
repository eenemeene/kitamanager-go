'use client';

import { useTranslations } from 'next-intl';
import { GripHorizontal } from 'lucide-react';
import { formatDate } from '@/lib/utils/formatting';
import type { BaseContract } from './timeline-utils';

interface BoundaryHandleProps {
  upperContract: BaseContract;
  lowerContract: BaseContract;
  boundaryIndex: number;
  onPointerDown: (boundaryIndex: number, e: React.PointerEvent) => void;
  isDragging: boolean;
  dragEndDate?: string;
  dragStartDate?: string;
}

export function BoundaryHandle({
  upperContract,
  lowerContract,
  boundaryIndex,
  onPointerDown,
  isDragging,
  dragEndDate,
  dragStartDate,
}: BoundaryHandleProps) {
  const t = useTranslations();

  // During drag, show the live dates; otherwise show the contract dates
  // The "upper" contract in the timeline (newer) has its `from` as the boundary start
  // The "lower" contract (older) has its `to` as the boundary end
  const endDate = dragEndDate || lowerContract.to || '';
  const startDate = dragStartDate || upperContract.from;

  return (
    <div
      data-testid="boundary-handle"
      role="slider"
      tabIndex={0}
      aria-label={t('timeline.dragToAdjust')}
      aria-valuenow={0}
      className={`group relative mx-4 flex cursor-ns-resize items-center justify-center rounded-md border-2 border-dashed py-2 transition-colors select-none ${
        isDragging
          ? 'border-primary bg-primary/10 cursor-grabbing'
          : 'hover:border-primary hover:bg-primary/5 border-muted-foreground/30'
      }`}
      onPointerDown={(e) => {
        e.preventDefault();
        onPointerDown(boundaryIndex, e);
      }}
      onKeyDown={(e) => {
        // Keyboard support: arrow keys shift boundary
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
          e.preventDefault();
          // Keyboard interaction is handled by the parent via a separate mechanism
        }
      }}
    >
      <GripHorizontal className="text-muted-foreground h-4 w-4" />
      <span className="text-muted-foreground ml-2 text-xs font-medium">
        {formatDate(endDate)} | {formatDate(startDate)}
      </span>
    </div>
  );
}
