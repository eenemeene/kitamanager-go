'use client';

import dynamic from 'next/dynamic';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';

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

export default function ChildrenStatisticsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();

  const { data: ageDistribution, isLoading: isLoadingAge } = useQuery({
    queryKey: queryKeys.statistics.ageDistribution(orgId),
    queryFn: () => apiClient.getAgeDistribution(orgId),
    enabled: !!orgId,
  });

  const { data: staffingHours, isLoading: isLoadingContracts } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(orgId),
    queryFn: () => apiClient.getStaffingHours(orgId),
    enabled: !!orgId,
  });

  const { data: contractProperties, isLoading: isLoadingContractProperties } = useQuery({
    queryKey: queryKeys.statistics.contractProperties(orgId),
    queryFn: () => apiClient.getContractPropertiesDistribution(orgId),
    enabled: !!orgId,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsChildren')}</h1>
      </div>

      {/* Monthly Contract Counts (full width) */}
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.childrenContractCount')}</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoadingContracts ? (
            <Skeleton className="h-[350px] w-full" />
          ) : staffingHours ? (
            <ChartErrorBoundary>
              <MonthlyContractChart data={staffingHours} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Age Distribution */}
        <Card>
          <CardHeader>
            <CardTitle>{t('statistics.ageDistribution')}</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoadingAge ? (
              <Skeleton className="h-[300px] w-full" />
            ) : ageDistribution ? (
              <ChartErrorBoundary>
                <AgeDistributionChart data={ageDistribution} />
              </ChartErrorBoundary>
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>

        {/* Contract Properties Distribution */}
        <Card>
          <CardHeader>
            <CardTitle>{t('statistics.contractProperties')}</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoadingContractProperties ? (
              <Skeleton className="h-[300px] w-full" />
            ) : contractProperties ? (
              <ChartErrorBoundary>
                <ContractPropertiesChart data={contractProperties} />
              </ChartErrorBoundary>
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
