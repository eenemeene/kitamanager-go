'use client';

import { useMemo } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { CircleDollarSign, Users, Baby, Table, Wallet, TrendingUp } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { FinancialSummaryCards } from '@/components/statistics/financial-summary-cards';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { getCurrentMonthStart } from '@/lib/utils/formatting';

export default function StatisticsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();

  const { data: financials, isLoading: isLoadingFinancials } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId),
    queryFn: () => apiClient.getFinancials(orgId),
    enabled: !!orgId,
  });

  // Get the current month's financial data point for summary cards
  const currentFinancials = useMemo(() => {
    if (!financials?.data_points?.length) return null;
    const currentMonth = getCurrentMonthStart();
    const exact = financials.data_points.find((dp) => dp.date === currentMonth);
    return exact ?? financials.data_points[financials.data_points.length - 1];
  }, [financials]);

  const subPages = [
    {
      href: `/organizations/${orgId}/statistics/financials`,
      icon: CircleDollarSign,
      title: t('nav.statisticsFinancials'),
      description: t('statistics.financialOverviewDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/staffing`,
      icon: Users,
      title: t('nav.statisticsStaffing'),
      description: t('statistics.staffingHoursDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/children`,
      icon: Baby,
      title: t('nav.statisticsChildren'),
      description: t('statistics.ageDistribution'),
    },
    {
      href: `/organizations/${orgId}/statistics/occupancy`,
      icon: Table,
      title: t('nav.statisticsOccupancy'),
      description: t('statistics.occupancyDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/budget`,
      icon: Wallet,
      title: t('nav.statisticsBudget'),
      description: t('statistics.budgetDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/forecast`,
      icon: TrendingUp,
      title: t('nav.statisticsForecast'),
      description: t('statistics.forecastDescription'),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('statistics.title')}</h1>
      </div>

      {/* Financial Summary Cards */}
      {isLoadingFinancials ? (
        <div className="grid gap-4 md:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <Card key={i}>
              <CardHeader className="pb-2">
                <Skeleton className="h-4 w-24" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-8 w-32" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : currentFinancials ? (
        <FinancialSummaryCards
          totalIncome={currentFinancials.total_income}
          totalExpenses={currentFinancials.total_expenses}
          balance={currentFinancials.balance}
        />
      ) : null}

      {/* Sub-page Link Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        {subPages.map((page) => {
          const Icon = page.icon;
          return (
            <Link key={page.href} href={page.href}>
              <Card className="hover:bg-muted/50 transition-colors">
                <CardHeader>
                  <div className="flex items-center gap-3">
                    <Icon className="text-muted-foreground h-5 w-5" />
                    <CardTitle className="text-base">{page.title}</CardTitle>
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-muted-foreground text-sm">{page.description}</p>
                </CardContent>
              </Card>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
