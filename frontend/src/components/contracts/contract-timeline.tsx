'use client';

import { useCallback, type ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import { parseISO, addDays } from 'date-fns';
import { getContractStatus } from '@/lib/utils/contracts';
import { formatDateForApi } from '@/lib/utils/formatting';
import type { ContractBatchUpdateItem } from '@/lib/api/types';
import { buildTimelineItems, type BaseContract } from './timeline-utils';
import { TimelineSegment } from './timeline-segment';
import { BoundaryHandle } from './boundary-handle';

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

  const handleBoundaryChange = useCallback(
    (upperContract: BaseContract, lowerContract: BaseContract, newTo: string, newFrom: string) => {
      const updates: ContractBatchUpdateItem[] = [
        { id: lowerContract.id, to: formatDateForApi(newTo) },
        { id: upperContract.id, from: formatDateForApi(newFrom) ?? undefined },
      ];
      onBoundaryChange(updates);
    },
    [onBoundaryChange]
  );

  if (contracts.length === 0) {
    return (
      <div data-testid="timeline-empty" className="text-muted-foreground py-8 text-center text-sm">
        {t('timeline.noContracts')}
      </div>
    );
  }

  const items = buildTimelineItems(contracts);

  return (
    <div
      data-testid="contract-timeline"
      className={`relative py-2 ${isUpdating ? 'pointer-events-none opacity-60' : ''}`}
    >
      {/* Vertical timeline line — aligned with dots (pl-3 + w-7/2 = 12px + 14px = 26px) */}
      <div className="bg-border absolute top-0 bottom-0 left-[1.625rem] w-px" />

      <div className="space-y-2">
        {items.map((item, i) => {
          if (item.type === 'segment') {
            const contract = contracts[item.index] as T;
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
            const upper = contracts[item.upperIndex];
            const lower = contracts[item.lowerIndex];
            const minDate = parseISO(lower.from);
            const maxDate = upper.to ? addDays(parseISO(upper.to), -1) : null;
            return (
              <BoundaryHandle
                key={`boundary-${item.upperIndex}-${item.lowerIndex}`}
                upperContract={upper}
                lowerContract={lower}
                minDate={minDate}
                maxDate={maxDate}
                onBoundaryChange={(newTo, newFrom) =>
                  handleBoundaryChange(upper, lower, newTo, newFrom)
                }
                isUpdating={isUpdating}
              />
            );
          }

          if (item.type === 'gap') {
            return (
              <div key={`gap-${i}`} className="relative flex gap-4 pl-3">
                {/* Gap dot */}
                <div className="relative flex w-7 shrink-0 items-center justify-center">
                  <div className="border-muted-foreground/40 z-10 h-2 w-2 rounded-full border bg-white dark:bg-gray-950" />
                </div>
                <div
                  data-testid="timeline-gap"
                  className="text-muted-foreground max-w-lg min-w-0 flex-1 py-1 text-center text-xs"
                >
                  {t('timeline.gap', { days: item.gapDays })}
                </div>
              </div>
            );
          }

          return null;
        })}
      </div>
    </div>
  );
}
