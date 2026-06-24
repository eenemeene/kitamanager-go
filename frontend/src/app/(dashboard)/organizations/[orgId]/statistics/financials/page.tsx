'use client';

import { useState, useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries } from '@tanstack/react-query';
import type { FundingComparisonResponse, FundingComparisonSummary } from '@/lib/api/types';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { CalculationWarningsBanner } from '@/components/charts/calculation-warnings-banner';
import { StatisticsPageHeader } from '@/components/statistics/statistics-page-header';
import { FinancialSummaryCards } from '@/components/statistics/financial-summary-cards';
import { BudgetTable } from '@/components/charts/budget-table';
import { YearStepper } from '@/components/ui/year-stepper';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { getCurrentMonthStart, toLocalDateString } from '@/lib/utils/formatting';

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

const FundingComparisonChart = dynamic(
  () =>
    import('@/components/charts/funding-comparison-chart').then(
      (mod) => mod.FundingComparisonChart
    ),
  { ssr: false, loading: () => <Skeleton className="h-[500px] w-full" /> }
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

  // Derive date range from financials data for compare queries (12-month windows)
  const compareWindows = useMemo(() => {
    const dps = financials?.data_points;
    if (!dps?.length) return [];
    // Only include months that have actual bills
    const billMonths = dps
      .filter((dp) => dp.actual_funding != null)
      .map((dp) => dp.date)
      .filter((d): d is string => d !== undefined);
    if (billMonths.length === 0) return [];
    const first = billMonths[0]!;
    const last = billMonths[billMonths.length - 1]!;
    // Split into 12-month windows
    const windows: { from: string; to: string }[] = [];
    let wFrom: string = first;
    while (wFrom <= last) {
      // Parse as local midnight and format with toLocalDateString so the
      // month arithmetic round-trips on the same calendar date. The old
      // `new Date(wFrom)` (parsed as UTC) + `.toISOString().slice(0,10)`
      // (formatted as UTC) mixed zones and shifted the date by a day in
      // behind-UTC locales.
      const fromDate = new Date(`${wFrom}T00:00:00`);
      const toDate = new Date(fromDate);
      toDate.setMonth(toDate.getMonth() + 11);
      const wToStr = toLocalDateString(toDate);
      const wTo = wToStr > last ? last : wToStr;
      windows.push({ from: wFrom, to: wTo });
      const nextDate = new Date(fromDate);
      nextDate.setMonth(nextDate.getMonth() + 12);
      wFrom = toLocalDateString(nextDate);
    }
    return windows;
  }, [financials]);

  const compareResults = useQueries({
    queries: compareWindows.map((w) => ({
      queryKey: queryKeys.governmentFundingBillPeriods.compareRange(orgId, w.from, w.to),
      queryFn: () => apiClient.compareBills(orgId, { from: w.from, to: w.to }),
      enabled: !!orgId && compareWindows.length > 0,
      staleTime: 5 * 60 * 1000,
    })),
  });

  const compareData = useMemo(() => {
    const map = new Map<string, FundingComparisonResponse>();
    for (const result of compareResults) {
      if (result.data) {
        for (const comp of result.data.comparisons) {
          map.set(comp.bill_from, comp);
        }
      }
    }
    return map.size > 0 ? map : undefined;
  }, [compareResults]);

  const compareSummaries = useMemo(() => {
    const map = new Map<string, FundingComparisonSummary>();
    for (let i = 0; i < compareResults.length; i++) {
      const result = compareResults[i];
      if (result.data?.summary && compareWindows[i]) {
        const w = compareWindows[i];
        map.set(`${w.from}:${w.to}`, result.data.summary);
      }
    }
    return map.size > 0 ? map : undefined;
  }, [compareResults, compareWindows]);

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
        descriptionKey="statistics.navFinancialsDescription"
        printHref={`/organizations/${orgId}/statistics/financials/print`}
      />

      {/* Data-quality warnings from the backend calculator (e.g.
          employees whose pay plan couldn't be resolved). The salary
          for those rows was excluded from the totals below; without
          this banner the lower-than-expected numbers would be
          unattributable. See backend F1 commit for the warning
          codes. */}
      <CalculationWarningsBanner warnings={financials?.warnings} />

      {/* Financial Summary Cards */}
      {currentFinancials && (
        <FinancialSummaryCards
          totalIncome={currentFinancials.total_income ?? 0}
          totalExpenses={currentFinancials.total_expenses ?? 0}
          balance={currentFinancials.balance ?? 0}
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

      {/* Actual vs Calculated Funding */}
      {financials?.data_points?.length && (
        <Card>
          <CardHeader>
            <CardTitle>{t('statistics.fundingActualVsCalculated')}</CardTitle>
            <p className="text-muted-foreground text-sm">
              {t('statistics.fundingActualVsCalculatedDescription')}
            </p>
          </CardHeader>
          <CardContent>
            <ChartErrorBoundary>
              <FundingComparisonChart
                data={financials}
                compareData={compareData}
                compareSummaries={compareSummaries}
              />
            </ChartErrorBoundary>
          </CardContent>
        </Card>
      )}

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
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground text-sm">{t('common.year')}</span>
            <YearStepper value={budgetYear} onChange={setBudgetYear} />
          </div>
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
