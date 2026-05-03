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
        className="text-muted-foreground hover:text-foreground inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md transition-colors"
        title={t('common.print')}
      >
        <Printer className="h-4 w-4" />
      </Link>
    </div>
  );
}
