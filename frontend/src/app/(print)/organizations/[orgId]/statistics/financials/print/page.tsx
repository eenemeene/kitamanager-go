'use client';

import { useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { BudgetTable } from '@/components/charts/budget-table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { formatCurrency, getCurrentMonthStart } from '@/lib/utils/formatting';
import { useUiStore } from '@/stores/ui-store';

const FinancialsChart = dynamic(
  () => import('@/components/charts/financials-bar-chart').then((mod) => mod.FinancialsChart),
  { ssr: false, loading: () => <Skeleton className="h-[580px] w-full" /> }
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

export default function FinancialsPrintPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { organizations, fetchOrganizations } = useUiStore();
  const [budgetYear] = useState(new Date().getFullYear());

  const budgetFrom = `${budgetYear}-01-01`;
  const budgetTo = `${budgetYear}-12-01`;

  useQuery({
    queryKey: ['organizations-load'],
    queryFn: async () => {
      if (organizations.length === 0) await fetchOrganizations();
      return null;
    },
  });

  const orgName = organizations.find((o) => o.id === orgId)?.name ?? '';

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
    <div className="mx-auto max-w-[1100px] p-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsFinancials')}</h1>
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

      {/* Financial Summary Cards */}
      {currentFinancials && (
        <div className="mb-8 grid break-inside-avoid grid-cols-3 gap-4">
          <div className="rounded-lg border p-4">
            <p className="text-muted-foreground text-sm font-medium">
              {t('statistics.totalIncome')}
            </p>
            <p className="mt-1 text-2xl font-bold text-green-600 dark:text-green-400">
              {formatCurrency(currentFinancials.total_income)}
            </p>
          </div>
          <div className="rounded-lg border p-4">
            <p className="text-muted-foreground text-sm font-medium">
              {t('statistics.totalExpenses')}
            </p>
            <p className="mt-1 text-2xl font-bold text-red-600 dark:text-red-400">
              {formatCurrency(currentFinancials.total_expenses)}
            </p>
          </div>
          <div className="rounded-lg border p-4">
            <p className="text-muted-foreground text-sm font-medium">{t('statistics.balance')}</p>
            <p
              className={`mt-1 text-2xl font-bold ${
                currentFinancials.balance >= 0
                  ? 'text-blue-600 dark:text-blue-400'
                  : 'text-red-600 dark:text-red-400'
              }`}
            >
              {formatCurrency(currentFinancials.balance)}
            </p>
          </div>
        </div>
      )}

      {/* Financial Overview Chart */}
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

      {/* Breakdown Pie Charts */}
      {currentFinancials && (
        <div className="mb-8 grid break-inside-avoid grid-cols-2 gap-6">
          <div>
            <h2 className="mb-3 text-xl font-semibold">{t('statistics.fundingBreakdown')}</h2>
            <ChartErrorBoundary>
              <FundingBreakdownChart data={currentFinancials} />
            </ChartErrorBoundary>
          </div>
          <div>
            <h2 className="mb-3 text-xl font-semibold">{t('statistics.expenseBreakdown')}</h2>
            <ChartErrorBoundary>
              <ExpenseBreakdownChart data={currentFinancials} />
            </ChartErrorBoundary>
          </div>
        </div>
      )}

      {/* Budget Table */}
      <div className="break-inside-avoid">
        <h2 className="mb-3 text-xl font-semibold">
          {t('statistics.budgetOverview')} &middot; {budgetYear}
        </h2>
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
