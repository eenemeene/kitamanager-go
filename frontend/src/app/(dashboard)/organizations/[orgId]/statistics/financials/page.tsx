'use client';

import { useState, useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { StatisticsPageHeader } from '@/components/statistics/statistics-page-header';
import { FinancialSummaryCards } from '@/components/statistics/financial-summary-cards';
import { BudgetTable } from '@/components/charts/budget-table';
import { YearStepper } from '@/components/ui/year-stepper';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { getCurrentMonthStart } from '@/lib/utils/formatting';

const FinancialsChart = dynamic(
  () => import('@/components/charts/financials-bar-chart').then((mod) => mod.FinancialsChart),
  { ssr: false, loading: () => <Skeleton className="h-[580px] w-full" /> }
);

const FinancialSummaryChart = dynamic(
  () =>
    import('@/components/charts/financial-summary-chart').then((mod) => mod.FinancialSummaryChart),
  { ssr: false, loading: () => <Skeleton className="h-[350px] w-full" /> }
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
  const [budgetYear, setBudgetYear] = useState(new Date().getFullYear());

  const budgetFrom = `${budgetYear}-01-01`;
  const budgetTo = `${budgetYear}-12-01`;

  const { data: financials, isLoading: isLoadingFinancials } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId),
    queryFn: () => apiClient.getFinancials(orgId),
    enabled: !!orgId,
  });

  const { data: budgetFinancials, isLoading: isLoadingBudget } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId, budgetFrom, budgetTo),
    queryFn: () => apiClient.getFinancials(orgId, { from: budgetFrom, to: budgetTo }),
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
      <StatisticsPageHeader
        titleKey="nav.statisticsFinancials"
        printHref={`/organizations/${orgId}/statistics/financials/print`}
      />

      {/* Financial Summary Cards */}
      {currentFinancials && (
        <FinancialSummaryCards
          totalIncome={currentFinancials.total_income}
          totalExpenses={currentFinancials.total_expenses}
          balance={currentFinancials.balance}
        />
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
            <Skeleton className="h-[580px] w-full" />
          ) : financials ? (
            <ChartErrorBoundary>
              <FinancialsChart data={financials} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>

      {/* Annual Summary Chart */}
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.financialSummary')}</CardTitle>
          <p className="text-muted-foreground text-sm">
            {t('statistics.financialSummaryDescription')}
          </p>
        </CardHeader>
        <CardContent>
          {isLoadingFinancials ? (
            <Skeleton className="h-[350px] w-full" />
          ) : financials ? (
            <ChartErrorBoundary>
              <FinancialSummaryChart data={financials} />
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

      {/* Budget Table */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div>
            <CardTitle>{t('statistics.budgetOverview')}</CardTitle>
            <p className="text-muted-foreground mt-1 text-sm">
              {t('statistics.budgetDescription')}
            </p>
          </div>
          <YearStepper value={budgetYear} onChange={setBudgetYear} />
        </CardHeader>
        <CardContent>
          {isLoadingBudget ? (
            <Skeleton className="h-[300px] w-full" />
          ) : budgetFinancials ? (
            <ChartErrorBoundary>
              <BudgetTable data={budgetFinancials} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
