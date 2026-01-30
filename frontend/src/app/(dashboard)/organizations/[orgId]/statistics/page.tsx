'use client';

import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { apiClient } from '@/lib/api/client';
import { AgeDistributionChart } from '@/components/charts/age-distribution-chart';
import { MonthlyContractChart } from '@/components/charts/monthly-contract-chart';

export default function StatisticsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();

  const { data: ageDistribution, isLoading: isLoadingAge } = useQuery({
    queryKey: ['age-distribution', orgId],
    queryFn: () => apiClient.getAgeDistribution(orgId),
    enabled: !!orgId,
  });

  const { data: contractCounts, isLoading: isLoadingContracts } = useQuery({
    queryKey: ['contract-counts', orgId],
    queryFn: () => apiClient.getChildrenContractCountByMonth(orgId),
    enabled: !!orgId,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('statistics.title')}</h1>
      </div>

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
              <AgeDistributionChart data={ageDistribution} />
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>

        {/* Monthly Contract Counts */}
        <Card>
          <CardHeader>
            <CardTitle>{t('statistics.childrenContractCount')}</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoadingContracts ? (
              <Skeleton className="h-[300px] w-full" />
            ) : contractCounts ? (
              <MonthlyContractChart data={contractCounts} />
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
