'use client';

import React from 'react';
import { useTranslations } from 'next-intl';
import { AlertTriangle } from 'lucide-react';
import { ErrorBoundary } from '@/components/error-boundary';

function ChartErrorFallback() {
  const t = useTranslations('statistics');
  return (
    <div className="flex h-[300px] items-center justify-center rounded-lg border border-dashed">
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <AlertTriangle className="h-4 w-4" />
        <span>{t('chartRenderError')}</span>
      </div>
    </div>
  );
}

export function ChartErrorBoundary({ children }: { children: React.ReactNode }) {
  return <ErrorBoundary fallback={<ChartErrorFallback />}>{children}</ErrorBoundary>;
}
