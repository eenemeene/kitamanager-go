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
  active: 'border-l-green-500 bg-green-50 dark:bg-green-950/20',
  upcoming: 'border-l-amber-500 bg-amber-50 dark:bg-amber-950/20',
  ended: 'border-l-gray-400 bg-gray-50 dark:bg-gray-900/20',
};

export function TimelineSegment({ from, to, status, children }: TimelineSegmentProps) {
  const t = useTranslations();

  return (
    <div
      data-testid="timeline-segment"
      data-status={status}
      className={`mx-4 rounded-md border-l-4 p-3 ${statusColors[status]}`}
    >
      <div className="text-muted-foreground mb-1 text-xs font-medium">
        {formatDate(from)} &mdash; {to ? formatDate(to) : t('common.ongoing')}
      </div>
      <div className="flex flex-wrap items-center gap-2">{children}</div>
    </div>
  );
}
