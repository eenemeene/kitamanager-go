'use client';

import { useMemo } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { OccupancyTable } from '@/components/charts/occupancy-table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { useUiStore } from '@/stores/ui-store';
import {
  parseReportMonth,
  formatReportMonthLong,
  formatKitaYearLabel,
} from '@/lib/utils/report-month';

export default function OccupancyPrintPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { organizations, fetchOrganizations } = useUiStore();
  const searchParams = useSearchParams();
  // The report month drives every date filter on this page so the rendered
  // matrix matches the page title. Falls back to the current calendar month
  // for dev / manual visits without ?month=.
  const reportMonth = useMemo(() => parseReportMonth(searchParams.get('month')), [searchParams]);

  useQuery({
    queryKey: ['organizations-load'],
    queryFn: async () => {
      if (organizations.length === 0) await fetchOrganizations();
      return null;
    },
  });

  const orgName = organizations.find((o) => o.id === orgId)?.name ?? '';

  // Annual matrix → Kita year (Aug → Jul) containing the report month.
  // German Kitas plan and report against the Kita year, not the calendar year.
  const { data: occupancy, isLoading } = useQuery({
    queryKey: queryKeys.statistics.occupancy(
      orgId,
      undefined,
      reportMonth.kitaYearFrom,
      reportMonth.kitaYearTo
    ),
    queryFn: () =>
      apiClient.getOccupancy(orgId, {
        from: reportMonth.kitaYearFrom,
        to: reportMonth.kitaYearTo,
      }),
    enabled: !!orgId,
  });

  return (
    <div className="mx-auto max-w-[1100px] p-8" data-print-ready={!isLoading ? 'true' : undefined}>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {t('nav.statisticsOccupancy')} &middot; {formatReportMonthLong(reportMonth)}
          </h1>
          <p className="text-muted-foreground mt-1 text-sm">
            {orgName} &middot; {new Date().toLocaleDateString()}
          </p>
        </div>
        <button
          className="no-print bg-primary text-primary-foreground hover:bg-primary/90 inline-flex h-10 items-center gap-2 rounded-md px-4 text-sm font-medium"
          onClick={() => window.print()}
        >
          <Printer className="h-4 w-4" />
          {t('common.print')}
        </button>
      </div>

      <div className="break-inside-avoid">
        <h2 className="mb-1 text-xl font-semibold">{t('statistics.occupancyMatrix')}</h2>
        <p className="text-muted-foreground mb-3 text-sm">{formatKitaYearLabel(reportMonth)}</p>
        {isLoading ? (
          <Skeleton className="h-[300px] w-full" />
        ) : occupancy ? (
          <ChartErrorBoundary>
            <OccupancyTable data={occupancy} />
          </ChartErrorBoundary>
        ) : null}
      </div>
    </div>
  );
}
