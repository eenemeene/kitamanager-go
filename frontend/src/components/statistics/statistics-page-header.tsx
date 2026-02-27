'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { Printer } from 'lucide-react';

interface StatisticsPageHeaderProps {
  titleKey: string;
  printHref: string;
}

export function StatisticsPageHeader({ titleKey, printHref }: StatisticsPageHeaderProps) {
  const t = useTranslations();

  return (
    <div className="flex items-center justify-between">
      <h1 className="text-3xl font-bold tracking-tight">{t(titleKey)}</h1>
      <Link
        href={printHref}
        target="_blank"
        className="text-muted-foreground hover:text-foreground inline-flex h-9 w-9 items-center justify-center rounded-md transition-colors"
        title={t('common.print')}
      >
        <Printer className="h-4 w-4" />
      </Link>
    </div>
  );
}
