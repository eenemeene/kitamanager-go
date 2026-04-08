'use client';

import { useTranslations } from 'next-intl';
import { formatDate } from '@/lib/utils/formatting';
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
  ended: 'border-gray-400 bg-gray-50 dark:bg-gray-900/20',
};

const dotColors = {
  active: 'bg-green-500',
  upcoming: 'bg-amber-500',
  ended: 'bg-gray-400',
};

export function TimelineSegment({ from, to, status, children }: TimelineSegmentProps) {
  const t = useTranslations();

  return (
    <div data-testid="timeline-segment" data-status={status} className="relative flex gap-4 pl-3">
      {/* Timeline dot */}
      <div className="relative flex w-7 shrink-0 items-center justify-center">
        <div
          className={`z-10 h-3 w-3 rounded-full ring-2 ring-white dark:ring-gray-950 ${dotColors[status]}`}
        />
      </div>
      {/* Content card */}
      <div className={`min-w-0 flex-1 rounded-md border-l-4 p-3 ${statusColors[status]}`}>
        <div className="text-muted-foreground mb-1 text-xs font-medium">
          {formatDate(from)} &mdash; {to ? formatDate(to) : t('common.ongoing')}
        </div>
        <div className="flex flex-wrap items-center gap-2">{children}</div>
      </div>
    </div>
  );
}
