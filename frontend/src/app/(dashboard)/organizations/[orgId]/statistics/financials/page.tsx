'use client';

import { useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { formatCurrency, getCurrentMonthStart } from '@/lib/utils/formatting';

const FinancialsChart = dynamic(
  () => import('@/components/charts/financials-bar-chart').then((mod) => mod.FinancialsChart),
  { ssr: false, loading: () => <Skeleton className="h-[400px] w-full" /> }
);

const FundingBreakdownChart = dynamic(
  () =>
    import('@/components/charts/funding-breakdown-chart').then((mod) => mod.FundingBreakdownChart),
  { ssr: false, loading: () => <Skeleton className="h-[350px] w-full" /> }
);

const ExpenseBreakdownChart = dynamic(
  () =>
    import('@/components/charts/expense-breakdown-chart').then((mod) => mod.ExpenseBreakdownChart),
  { ssr: false, loading: () => <Skeleton className="h-[350px] w-full" /> }
);

export default function FinancialsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();

  const { data: financials, isLoading: isLoadingFinancials } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId),
    queryFn: () => apiClient.getFinancials(orgId),
    enabled: !!orgId,
  });

  const currentFinancials = useMemo(() => {
    if (!financials?.data_points?.length) return null;
    const currentMonth = getCurrentMonthStart();
    const exact = financials.data_points.find((dp) => dp.date === currentMonth);
    return exact ?? financials.data_points[financials.data_points.length - 1];
  }, [financials]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsFinancials')}</h1>
      </div>

      {/* Financial Summary Cards */}
      {currentFinancials && (
        <div className="grid gap-4 md:grid-cols-3">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-muted-foreground text-sm font-medium">
                {t('statistics.totalIncome')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-600 dark:text-green-400">
                {formatCurrency(currentFinancials.total_income)}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-muted-foreground text-sm font-medium">
                {t('statistics.totalExpenses')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-red-600 dark:text-red-400">
                {formatCurrency(currentFinancials.total_expenses)}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-muted-foreground text-sm font-medium">
                {t('statistics.balance')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div
                className={`text-2xl font-bold ${
                  currentFinancials.balance >= 0
                    ? 'text-blue-600 dark:text-blue-400'
                    : 'text-red-600 dark:text-red-400'
                }`}
              >
                {formatCurrency(currentFinancials.balance)}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Financial Overview Chart */}
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.financialOverview')}</CardTitle>
          <p className="text-muted-foreground text-sm">
            {t('statistics.financialOverviewDescription')}
          </p>
        </CardHeader>
        <CardContent>
          {isLoadingFinancials ? (
            <Skeleton className="h-[400px] w-full" />
          ) : financials ? (
            <ChartErrorBoundary>
              <FinancialsChart data={financials} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>

      {/* Breakdown Pie Charts */}
      {currentFinancials && (
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>{t('statistics.fundingBreakdown')}</CardTitle>
              <CardDescription>{t('statistics.fundingBreakdownDescription')}</CardDescription>
            </CardHeader>
            <CardContent>
              <ChartErrorBoundary>
                <FundingBreakdownChart data={currentFinancials} />
              </ChartErrorBoundary>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>{t('statistics.expenseBreakdown')}</CardTitle>
              <CardDescription>{t('statistics.expenseBreakdownDescription')}</CardDescription>
            </CardHeader>
            <CardContent>
              <ChartErrorBoundary>
                <ExpenseBreakdownChart data={currentFinancials} />
              </ChartErrorBoundary>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
