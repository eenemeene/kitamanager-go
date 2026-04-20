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
  active: 'border-success bg-success/10',
  upcoming: 'border-warning bg-warning/10',
  ended: 'border-muted-foreground/30 bg-muted/40',
};

const dotColors = {
  active: 'bg-success',
  upcoming: 'bg-warning',
  ended: 'bg-muted-foreground/40',
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
