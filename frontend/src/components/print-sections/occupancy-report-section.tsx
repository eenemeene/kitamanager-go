'use client';

import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { OccupancyTable } from '@/components/charts/occupancy-table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { type ReportMonth, formatKitaYearLabel } from '@/lib/utils/report-month';
import { useFormatters } from '@/hooks/use-formatters';

interface Props {
  orgId: number;
  reportMonth: ReportMonth;
}

/**
 * Occupancy section of a print report. Renders the annual matrix
 * scoped to the Kita year containing the report month. Owns its
 * own data fetches so the same component can drop into the
 * standalone /occupancy/print page and (in a follow-up) into the
 * combined /report/print page.
 *
 * Loading is intentionally not exposed via a callback prop — the
 * host page observes "all sections settled" via TanStack Query's
 * useIsFetching(), which works the same whether one or many
 * sections render on the page.
 */
export function OccupancyReportSection({ orgId, reportMonth }: Props) {
  const t = useTranslations();

  const fmt = useFormatters();
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
    <section className="report-section break-inside-avoid">
      <h2 className="mb-1 text-xl font-semibold">{t('statistics.occupancyMatrix')}</h2>
      <p className="text-muted-foreground mb-3 text-sm">
        {formatKitaYearLabel(reportMonth, fmt.locale)}
      </p>
      {isLoading ? (
        <Skeleton className="h-[300px] w-full" />
      ) : occupancy ? (
        <ChartErrorBoundary>
          <OccupancyTable data={occupancy} />
        </ChartErrorBoundary>
      ) : null}
    </section>
  );
}
