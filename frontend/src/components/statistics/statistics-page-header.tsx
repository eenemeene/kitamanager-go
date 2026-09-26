'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { Printer } from 'lucide-react';

interface StatisticsPageHeaderProps {
  titleKey: string;
  /** Optional i18n key for a one-line description shown under the title */
  descriptionKey?: string;
  printHref: string;
}

export function StatisticsPageHeader({
  titleKey,
  descriptionKey,
  printHref,
}: StatisticsPageHeaderProps) {
  const t = useTranslations();

  return (
    <div className="flex items-start justify-between gap-3">
      <div className="min-w-0">
        <h1 className="text-3xl font-bold tracking-tight">{t(titleKey)}</h1>
        {descriptionKey && (
          <p className="text-muted-foreground mt-1 max-w-3xl text-sm">{t(descriptionKey)}</p>
        )}
      </div>
      <Link
        href={printHref}
        target="_blank"
        // Full touch target, compact only from lg: where there is a mouse --
        // the same shape app-sidebar's submenu chevron uses. It was 36px at
        // every width, on four pages a manager reads from a tablet.
        className="text-muted-foreground hover:text-foreground inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-md transition-colors lg:h-9 lg:w-9"
        title={t('common.print')}
      >
        <Printer className="h-4 w-4" />
      </Link>
    </div>
  );
}
