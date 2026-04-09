'use client';

import { useTranslations } from 'next-intl';
import { formatDate } from '@/lib/utils/formatting';
import { Badge } from '@/components/ui/badge';
import type { ReactNode } from 'react';

interface TimelineSegmentProps {
  from: string;
  to?: string | null;
  status: 'active' | 'upcoming' | 'ended';
  children: ReactNode;
}

const statusColors = {
  active: 'border-green-500 bg-green-50 dark:bg-green-950/20',
  upcoming: 'border-amber-500 bg-amber-50 dark:bg-amber-950/20',
  ended: 'border-gray-300 bg-gray-50 dark:border-gray-600 dark:bg-gray-900/20',
};

const dotColors = {
  active: 'bg-green-500',
  upcoming: 'bg-amber-500',
  ended: 'bg-gray-400',
};

const badgeVariants = {
  active: 'success' as const,
  upcoming: 'warning' as const,
  ended: 'secondary' as const,
};

export function TimelineSegment({ from, to, status, children }: TimelineSegmentProps) {
  const t = useTranslations();

  const statusLabel =
    status === 'active'
      ? t('common.active')
      : status === 'upcoming'
        ? t('common.upcoming')
        : t('common.ended');

  return (
    <div data-testid="timeline-segment" data-status={status} className="relative flex gap-4 pl-3">
      {/* Timeline dot */}
      <div className="relative flex w-7 shrink-0 justify-center pt-4">
        <div
          className={`z-10 h-3 w-3 rounded-full ring-2 ring-white dark:ring-gray-950 ${dotColors[status]}`}
        />
      </div>
      {/* Content card */}
      <div
        className={`max-w-lg min-w-0 flex-1 rounded-lg border-l-4 p-3 shadow-sm ${statusColors[status]}`}
      >
        {/* Header row: status badge + date range */}
        <div className="mb-2 flex flex-wrap items-center gap-2">
          <Badge variant={badgeVariants[status]} className="text-xs">
            {statusLabel}
          </Badge>
          <span className="text-muted-foreground text-xs">
            {formatDate(from)} &mdash; {to ? formatDate(to) : t('common.ongoing')}
          </span>
        </div>
        {/* Details */}
        <div className="flex flex-wrap items-center gap-2">{children}</div>
      </div>
    </div>
  );
}
