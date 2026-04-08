'use client';

import { useCallback, type ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import { getContractStatus } from '@/lib/utils/contracts';
import type { ContractBatchUpdateItem } from '@/lib/api/types';
import { buildTimelineItems, applyDragToContracts, type BaseContract } from './timeline-utils';
import { TimelineSegment } from './timeline-segment';
import { BoundaryHandle } from './boundary-handle';
import { useBoundaryDrag } from './use-boundary-drag';

interface ContractTimelineProps<T extends BaseContract> {
  contracts: T[];
  renderSegmentContent: (contract: T) => ReactNode;
  onBoundaryChange: (updates: ContractBatchUpdateItem[]) => Promise<unknown>;
  isUpdating?: boolean;
}

export function ContractTimeline<T extends BaseContract>({
  contracts,
  renderSegmentContent,
  onBoundaryChange,
  isUpdating,
}: ContractTimelineProps<T>) {
  const t = useTranslations();

  const handleDragEnd = useCallback(
    (updates: ContractBatchUpdateItem[]) => {
      onBoundaryChange(updates);
    },
    [onBoundaryChange]
  );

  const { dragState, handlePointerDown } = useBoundaryDrag({
    contracts,
    onDragEnd: handleDragEnd,
  });

  if (contracts.length === 0) {
    return (
      <div data-testid="timeline-empty" className="text-muted-foreground py-8 text-center text-sm">
        {t('timeline.noContracts')}
      </div>
    );
  }

  // Apply optimistic drag state to contracts for visual preview
  const displayContracts = dragState
    ? applyDragToContracts(
        contracts,
        dragState.upperIndex,
        dragState.lowerIndex,
        dragState.newTo,
        dragState.newFrom
      )
    : contracts;

  const items = buildTimelineItems(displayContracts);

  return (
    <div
      data-testid="contract-timeline"
      className={`relative space-y-1 py-2 ${isUpdating ? 'pointer-events-none opacity-60' : ''}`}
    >
      {/* Vertical timeline line */}
      <div className="bg-border absolute top-0 bottom-0 left-6 w-px" />

      {items.map((item, i) => {
        if (item.type === 'segment') {
          const contract = displayContracts[item.index] as T;
          const status = getContractStatus(contract) ?? 'ended';
          return (
            <TimelineSegment
              key={`seg-${contract.id}`}
              from={contract.from}
              to={contract.to}
              status={status}
            >
              {renderSegmentContent(contract)}
            </TimelineSegment>
          );
        }

        if (item.type === 'boundary') {
          const upper = displayContracts[item.upperIndex];
          const lower = displayContracts[item.lowerIndex];
          const isDragging = dragState?.boundaryIndex === item.upperIndex;
          return (
            <BoundaryHandle
              key={`boundary-${item.upperIndex}-${item.lowerIndex}`}
              upperContract={upper}
              lowerContract={lower}
              boundaryIndex={item.upperIndex}
              onPointerDown={handlePointerDown}
              isDragging={isDragging}
              dragEndDate={isDragging ? dragState?.newTo : undefined}
              dragStartDate={isDragging ? dragState?.newFrom : undefined}
            />
          );
        }

        if (item.type === 'gap') {
          return (
            <div
              key={`gap-${i}`}
              data-testid="timeline-gap"
              className="text-muted-foreground mx-4 flex items-center justify-center border-y border-dashed py-2 text-xs"
            >
              {t('timeline.gap', { days: item.gapDays })}
            </div>
          );
        }

        return null;
      })}
    </div>
  );
}
