'use client';

import { useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries } from '@tanstack/react-query';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { BudgetTable } from '@/components/charts/budget-table';
import type { FundingComparisonResponse, FundingComparisonSummary } from '@/lib/api/types';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { toLocalDateString } from '@/lib/utils/formatting';
import {
  type ReportMonth,
  formatReportMonthLong,
  formatKitaYearLabel,
} from '@/lib/utils/report-month';
import { useFormatters } from '@/hooks/use-formatters';

const FinancialsChart = dynamic(
  () => import('@/components/charts/financials-bar-chart').then((mod) => mod.FinancialsChart),
  { ssr: false, loading: () => <Skeleton className="h-[580px] w-full" /> }
);

const FundingBreakdownChart = dynamic(
  () =>
    import('@/components/charts/funding-breakdown-chart').then((mod) => mod.FundingBreakdownChart),
  { ssr: false, loading: () => <Skeleton className="h-[350px] w-full" /> }
);

const FinancialSummaryChart = dynamic(
  () =>
    import('@/components/charts/financial-summary-chart').then((mod) => mod.FinancialSummaryChart),
  { ssr: false, loading: () => <Skeleton className="h-[550px] w-full" /> }
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

interface Props {
  orgId: number;
  reportMonth: ReportMonth;
}

/**
 * Financials section of a print report. Six visualizations:
 *   - Summary cards for the report month (income / expenses / balance)
 *   - Multi-year overview bar chart
 *   - Actual-vs-calculated funding comparison
 *   - Cumulative balance
 *   - Funding + expense breakdown pies (the report month)
 *   - Annual budget table (Kita year)
 */
export function FinancialsReportSection({ orgId, reportMonth }: Props) {
  const t = useTranslations();

  const fmt = useFormatters();
  const { data: financials, isLoading: isLoadingFinancials } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId, reportMonth.trendFrom, reportMonth.trendTo),
    queryFn: () =>
      apiClient.getFinancials(orgId, {
        from: reportMonth.trendFrom,
        to: reportMonth.trendTo,
      }),
    enabled: !!orgId,
  });

  const { data: budgetFinancials, isLoading: isLoadingBudget } = useQuery({
    queryKey: queryKeys.statistics.financials(
      orgId,
      reportMonth.kitaYearFrom,
      reportMonth.kitaYearTo
    ),
    queryFn: () =>
      apiClient.getFinancials(orgId, {
        from: reportMonth.kitaYearFrom,
        to: reportMonth.kitaYearTo,
      }),
    enabled: !!orgId,
  });

  // compareWindows / compareResults / compareData / compareSummaries:
  // chunk the bill-month range from the financials response into
  // 12-month slices for the actual-vs-calculated comparison.
  const compareWindows = useMemo(() => {
    const dps = financials?.data_points;
    if (!dps?.length) return [];
    const billMonths: string[] = dps
      .filter((dp): dp is typeof dp & { date: string } => dp.actual_funding != null && !!dp.date)
      .map((dp) => dp.date);
    if (billMonths.length === 0) return [];
    const first = billMonths[0]!;
    const last = billMonths[billMonths.length - 1]!;
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

  // Pick the data point for the report month — replaces the previous
  // getCurrentMonthStart() drift bug that always picked wall-clock now.
  const reportMonthFinancials = useMemo(() => {
    if (!financials?.data_points?.length) return null;
    const exact = financials.data_points.find((dp) => dp.date === reportMonth.asOf);
    return exact ?? financials.data_points[financials.data_points.length - 1];
  }, [financials, reportMonth.asOf]);

  return (
    <section className="report-section">
      {reportMonthFinancials && (
        <div className="mb-8 break-inside-avoid">
          <p className="text-muted-foreground mb-2 text-sm">
            {formatReportMonthLong(reportMonth, fmt.locale)}
          </p>
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border p-4">
              <p className="text-muted-foreground text-sm font-medium">
                {t('statistics.totalIncome')}
              </p>
              <p className="text-success mt-1 text-2xl font-bold">
                {fmt.currency(reportMonthFinancials.total_income)}
              </p>
            </div>
            <div className="rounded-lg border p-4">
              <p className="text-muted-foreground text-sm font-medium">
                {t('statistics.totalExpenses')}
              </p>
              <p className="text-destructive mt-1 text-2xl font-bold">
                {fmt.currency(reportMonthFinancials.total_expenses)}
              </p>
            </div>
            <div className="rounded-lg border p-4">
              <p className="text-muted-foreground text-sm font-medium">{t('statistics.balance')}</p>
              <p
                className={`mt-1 text-2xl font-bold ${
                  (reportMonthFinancials.balance ?? 0) >= 0 ? 'text-info' : 'text-destructive'
                }`}
              >
                {fmt.currency(reportMonthFinancials.balance ?? 0)}
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="mb-8 break-inside-avoid">
        <h2 className="mb-3 text-xl font-semibold">{t('statistics.financialOverview')}</h2>
        {isLoadingFinancials ? (
          <Skeleton className="h-[580px] w-full" />
        ) : financials ? (
          <ChartErrorBoundary>
            <FinancialsChart data={financials} />
          </ChartErrorBoundary>
        ) : null}
      </div>

      {financials?.data_points?.length && (
        <div className="mb-8 break-inside-avoid">
          <h2 className="mb-3 text-xl font-semibold">
            {t('statistics.fundingActualVsCalculated')}
          </h2>
          <ChartErrorBoundary>
            <FundingComparisonChart
              data={financials}
              compareData={compareData}
              compareSummaries={compareSummaries}
              forceExpanded
            />
          </ChartErrorBoundary>
        </div>
      )}

      <div className="mb-8 break-inside-avoid">
        <h2 className="mb-3 text-xl font-semibold">{t('statistics.financialSummary')}</h2>
        <p className="text-muted-foreground mb-2 text-sm">
          {t('statistics.financialSummaryDescription')}
        </p>
        {isLoadingFinancials ? (
          <Skeleton className="h-[550px] w-full" />
        ) : financials ? (
          <ChartErrorBoundary>
            <FinancialSummaryChart data={financials} />
          </ChartErrorBoundary>
        ) : null}
      </div>

      {reportMonthFinancials && (
        <div className="mb-8 break-inside-avoid">
          <p className="text-muted-foreground mb-2 text-sm">
            {formatReportMonthLong(reportMonth, fmt.locale)}
          </p>
          <div className="grid grid-cols-2 gap-6">
            <div>
              <h2 className="mb-3 text-xl font-semibold">{t('statistics.fundingBreakdown')}</h2>
              <ChartErrorBoundary>
                <FundingBreakdownChart data={reportMonthFinancials} />
              </ChartErrorBoundary>
            </div>
            <div>
              <h2 className="mb-3 text-xl font-semibold">{t('statistics.expenseBreakdown')}</h2>
              <ChartErrorBoundary>
                <ExpenseBreakdownChart data={reportMonthFinancials} />
              </ChartErrorBoundary>
            </div>
          </div>
        </div>
      )}

      <div className="break-inside-avoid">
        <h2 className="mb-1 text-xl font-semibold">{t('statistics.budgetOverview')}</h2>
        <p className="text-muted-foreground mb-3 text-sm">
          {formatKitaYearLabel(reportMonth, fmt.locale)}
        </p>
        {isLoadingBudget ? (
          <Skeleton className="h-[300px] w-full" />
        ) : budgetFinancials ? (
          <ChartErrorBoundary>
            <BudgetTable data={budgetFinancials} />
          </ChartErrorBoundary>
        ) : null}
      </div>
    </section>
  );
}
