'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { CircleDollarSign, Users, Baby, Table, Wallet, TrendingUp } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function StatisticsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();

  const subPages = [
    {
      href: `/organizations/${orgId}/statistics/financials`,
      icon: CircleDollarSign,
      title: t('nav.statisticsFinancials'),
      description: t('statistics.navFinancialsDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/staffing`,
      icon: Users,
      title: t('nav.statisticsStaffing'),
      description: t('statistics.navStaffingDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/children`,
      icon: Baby,
      title: t('nav.statisticsChildren'),
      description: t('statistics.navChildrenDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/occupancy`,
      icon: Table,
      title: t('nav.statisticsOccupancy'),
      description: t('statistics.navOccupancyDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/budget`,
      icon: Wallet,
      title: t('nav.statisticsBudget'),
      description: t('statistics.navBudgetDescription'),
    },
    {
      href: `/organizations/${orgId}/statistics/forecast`,
      icon: TrendingUp,
      title: t('nav.statisticsForecast'),
      description: t('statistics.navForecastDescription'),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('statistics.title')}</h1>
      </div>

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
