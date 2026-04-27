'use client';

import { useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useParams, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { useUiStore } from '@/stores/ui-store';
import { parseReportMonth, formatReportMonthLong } from '@/lib/utils/report-month';

const AgeDistributionChart = dynamic(
  () =>
    import('@/components/charts/age-distribution-chart').then((mod) => mod.AgeDistributionChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

const MonthlyContractChart = dynamic(
  () =>
    import('@/components/charts/monthly-contract-chart').then((mod) => mod.MonthlyContractChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

const ContractPropertiesChart = dynamic(
  () =>
    import('@/components/charts/contract-properties-chart').then(
      (mod) => mod.ContractPropertiesChart
    ),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

export default function ChildrenPrintPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { organizations, fetchOrganizations } = useUiStore();
  const searchParams = useSearchParams();
  const reportMonth = useMemo(() => parseReportMonth(searchParams.get('month')), [searchParams]);

  useQuery({
    queryKey: ['organizations-load'],
    queryFn: async () => {
      if (organizations.length === 0) await fetchOrganizations();
      return null;
    },
  });

  const orgName = organizations.find((o) => o.id === orgId)?.name ?? '';

  // Snapshot at the report month — replaces the previous hard-coded
  // "${year}-06-01" June snapshot, which was always mid-year regardless
  // of when the report was generated.
  const { data: ageDistribution, isLoading: isLoadingAge } = useQuery({
    queryKey: queryKeys.statistics.ageDistribution(orgId),
    queryFn: () => apiClient.getAgeDistribution(orgId, reportMonth.asOf),
    enabled: !!orgId,
  });

  const { data: contractProperties, isLoading: isLoadingContractProperties } = useQuery({
    queryKey: queryKeys.statistics.contractProperties(orgId),
    queryFn: () => apiClient.getContractPropertiesDistribution(orgId, reportMonth.asOf),
    enabled: !!orgId,
  });

  // Multi-year trend chart: previous + current + next Kita year. Explicitly
  // pass the trend window so the chart matches the page title — without it,
  // the API returns its own ~25-month default which drifts as wall-clock time
  // moves past the report month.
  const { data: contractTrend, isLoading: isLoadingContracts } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(
      orgId,
      undefined,
      reportMonth.trendFrom,
      reportMonth.trendTo
    ),
    queryFn: () =>
      apiClient.getStaffingHours(orgId, {
        from: reportMonth.trendFrom,
        to: reportMonth.trendTo,
      }),
    enabled: !!orgId,
  });

  // Occupancy is consumed by MonthlyContractChart for the age-group legend
  // and per-month tooltip breakdown. Fetch over the same trend window as
  // contractTrend so the tooltips have data for every column the chart renders.
  const { data: occupancy } = useQuery({
    queryKey: queryKeys.statistics.occupancy(
      orgId,
      undefined,
      reportMonth.trendFrom,
      reportMonth.trendTo
    ),
    queryFn: () =>
      apiClient.getOccupancy(orgId, {
        from: reportMonth.trendFrom,
        to: reportMonth.trendTo,
      }),
    enabled: !!orgId,
  });

  return (
    <div
      className="mx-auto max-w-[1100px] p-8"
      data-print-ready={
        !isLoadingAge && !isLoadingContracts && !isLoadingContractProperties ? 'true' : undefined
      }
    >
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {t('nav.statisticsChildren')} &middot; {formatReportMonthLong(reportMonth)}
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

      {/* Monthly Contract Counts */}
      <div className="mb-8 break-inside-avoid">
        <h2 className="mb-3 text-xl font-semibold">{t('statistics.childrenContractCount')}</h2>
        {isLoadingContracts ? (
          <Skeleton className="h-[350px] w-full" />
        ) : contractTrend ? (
          <ChartErrorBoundary>
            <MonthlyContractChart data={contractTrend} occupancy={occupancy} />
          </ChartErrorBoundary>
        ) : null}
      </div>

      {/* Age Distribution & Contract Properties */}
      <div className="grid grid-cols-2 gap-6">
        <div className="break-inside-avoid">
          <h2 className="mb-3 text-xl font-semibold">{t('statistics.ageDistribution')}</h2>
          {isLoadingAge ? (
            <Skeleton className="h-[300px] w-full" />
          ) : ageDistribution ? (
            <ChartErrorBoundary>
              <AgeDistributionChart data={ageDistribution} />
            </ChartErrorBoundary>
          ) : null}
        </div>
        <div className="break-inside-avoid">
          <h2 className="mb-3 text-xl font-semibold">{t('statistics.contractProperties')}</h2>
          {isLoadingContractProperties ? (
            <Skeleton className="h-[300px] w-full" />
          ) : contractProperties ? (
            <ChartErrorBoundary>
              <ContractPropertiesChart data={contractProperties} />
            </ChartErrorBoundary>
          ) : null}
        </div>
      </div>
    </div>
  );
}
