'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { BudgetTable } from '@/components/charts/budget-table';
import { YearStepper } from '@/components/ui/year-stepper';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';

export default function BudgetPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const [year, setYear] = useState(new Date().getFullYear());

  const from = `${year}-01-01`;
  const to = `${year}-12-01`;

  const { data: financials, isLoading } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId, from, to),
    queryFn: () => apiClient.getFinancials(orgId, { from, to }),
    enabled: !!orgId,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsBudget')}</h1>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div>
            <CardTitle>{t('statistics.budgetOverview')}</CardTitle>
            <p className="text-muted-foreground mt-1 text-sm">
              {t('statistics.budgetDescription')}
            </p>
          </div>
          <YearStepper value={year} onChange={setYear} />
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Skeleton className="h-[300px] w-full" />
          ) : financials ? (
            <ChartErrorBoundary>
              <BudgetTable data={financials} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
