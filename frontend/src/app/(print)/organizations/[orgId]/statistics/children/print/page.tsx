'use client';

import dynamic from 'next/dynamic';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { useUiStore } from '@/stores/ui-store';

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

  useQuery({
    queryKey: ['organizations-load'],
    queryFn: async () => {
      if (organizations.length === 0) await fetchOrganizations();
      return null;
    },
  });

  const orgName = organizations.find((o) => o.id === orgId)?.name ?? '';

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
    <div className="mx-auto max-w-[1100px] p-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsChildren')}</h1>
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
        ) : staffingHours ? (
          <ChartErrorBoundary>
            <MonthlyContractChart data={staffingHours} />
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
