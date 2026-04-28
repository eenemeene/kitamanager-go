'use client';

import dynamic from 'next/dynamic';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { type ReportMonth } from '@/lib/utils/report-month';

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

interface Props {
  orgId: number;
  reportMonth: ReportMonth;
}

/**
 * Children section of a print report. Three visualizations:
 *   - Monthly contract count (multi-year trend, prev+current+next Kita year)
 *   - Age distribution (snapshot at the report month)
 *   - Contract properties (snapshot at the report month)
 *
 * The MonthlyContractChart consumes both staffing-hours data (for the
 * count timeline) and occupancy data (for the age-group legend +
 * tooltip breakdown). Both are fetched here.
 */
export function ChildrenReportSection({ orgId, reportMonth }: Props) {
  const t = useTranslations();

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
    <section className="report-section">
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
    </section>
  );
}
