'use client';

import { useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useParams, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { BudgetTable } from '@/components/charts/budget-table';
import type { FundingComparisonResponse, FundingComparisonSummary } from '@/lib/api/types';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { formatCurrency } from '@/lib/utils/formatting';
import { useUiStore } from '@/stores/ui-store';
import {
  parseReportMonth,
  formatReportMonthLong,
  formatKitaYearLabel,
} from '@/lib/utils/report-month';

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

export default function FinancialsPrintPage() {
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

  // Multi-year trend chart data — used by the bar chart, funding-comparison
  // chart, and cumulative balance chart. Spans previous + current + next
  // Kita year so the trends have history and forecast around the report month.
  const { data: financials, isLoading: isLoadingFinancials } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId, reportMonth.trendFrom, reportMonth.trendTo),
    queryFn: () =>
      apiClient.getFinancials(orgId, {
        from: reportMonth.trendFrom,
        to: reportMonth.trendTo,
      }),
    enabled: !!orgId,
  });

  // Budget table — Kita year (Aug → Jul) containing the report month.
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

  // Compare-bills windows: chunk the bill-month range from the financials
  // response into 12-month slices for the actual-vs-calculated comparison.
  // Logic unchanged from the year-based version; the windowing is driven by
  // which months have actual_funding present in the data.
  const compareWindows = useMemo(() => {
    const dps = financials?.data_points;
    if (!dps?.length) return [];
    const billMonths = dps.filter((dp) => dp.actual_funding != null).map((dp) => dp.date);
    if (billMonths.length === 0) return [];
    const first = billMonths[0];
    const last = billMonths[billMonths.length - 1];
    const windows: { from: string; to: string }[] = [];
    let wFrom = first;
    while (wFrom <= last) {
      const fromDate = new Date(wFrom);
      const toDate = new Date(fromDate);
      toDate.setMonth(toDate.getMonth() + 11);
      const wTo =
        toDate.toISOString().slice(0, 10) > last ? last : toDate.toISOString().slice(0, 10);
      windows.push({ from: wFrom, to: wTo });
      const nextDate = new Date(fromDate);
      nextDate.setMonth(nextDate.getMonth() + 12);
      wFrom = nextDate.toISOString().slice(0, 10);
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

  const isLoadingCompare = compareResults.some((r) => r.isLoading);

  // Pick the data point for the report month — replaces the previous
  // getCurrentMonthStart() call which always returned the calendar
  // current month regardless of report period (so a 2024 report
  // generated in 2026 would show April 2026 numbers in the cards).
  const reportMonthFinancials = useMemo(() => {
    if (!financials?.data_points?.length) return null;
    const exact = financials.data_points.find((dp) => dp.date === reportMonth.asOf);
    return exact ?? financials.data_points[financials.data_points.length - 1];
  }, [financials, reportMonth.asOf]);

  return (
    <div
      className="mx-auto max-w-[1100px] p-8"
      data-print-ready={
        !isLoadingFinancials && !isLoadingBudget && !isLoadingCompare ? 'true' : undefined
      }
    >
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {t('nav.statisticsFinancials')} &middot; {formatReportMonthLong(reportMonth)}
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

      {/* Summary cards — for the report month */}
      {reportMonthFinancials && (
        <div className="mb-8 break-inside-avoid">
          <p className="text-muted-foreground mb-2 text-sm">{formatReportMonthLong(reportMonth)}</p>
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border p-4">
              <p className="text-muted-foreground text-sm font-medium">
                {t('statistics.totalIncome')}
              </p>
              <p className="text-success mt-1 text-2xl font-bold">
                {formatCurrency(reportMonthFinancials.total_income)}
              </p>
            </div>
            <div className="rounded-lg border p-4">
              <p className="text-muted-foreground text-sm font-medium">
                {t('statistics.totalExpenses')}
              </p>
              <p className="text-destructive mt-1 text-2xl font-bold">
                {formatCurrency(reportMonthFinancials.total_expenses)}
              </p>
            </div>
            <div className="rounded-lg border p-4">
              <p className="text-muted-foreground text-sm font-medium">{t('statistics.balance')}</p>
              <p
                className={`mt-1 text-2xl font-bold ${
                  reportMonthFinancials.balance >= 0 ? 'text-info' : 'text-destructive'
                }`}
              >
                {formatCurrency(reportMonthFinancials.balance)}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Financial Overview Chart — multi-year trend */}
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

      {/* Actual vs Calculated Funding — multi-year trend */}
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

      {/* Cumulative Balance — multi-year trend */}
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

      {/* Breakdown Pie Charts — for the report month */}
      {reportMonthFinancials && (
        <div className="mb-8 break-inside-avoid">
          <p className="text-muted-foreground mb-2 text-sm">{formatReportMonthLong(reportMonth)}</p>
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

      {/* Budget Table — Kita year */}
      <div className="break-inside-avoid">
        <h2 className="mb-1 text-xl font-semibold">{t('statistics.budgetOverview')}</h2>
        <p className="text-muted-foreground mb-3 text-sm">{formatKitaYearLabel(reportMonth)}</p>
        {isLoadingBudget ? (
          <Skeleton className="h-[300px] w-full" />
        ) : budgetFinancials ? (
          <ChartErrorBoundary>
            <BudgetTable data={budgetFinancials} />
          </ChartErrorBoundary>
        ) : null}
      </div>
    </div>
  );
}
