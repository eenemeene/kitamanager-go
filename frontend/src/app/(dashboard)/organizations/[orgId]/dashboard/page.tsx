'use client';

import { useMemo } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Users, Baby, Clock, Upload } from 'lucide-react';
import { StatCard } from '@/components/dashboard/stat-card';
import { StepPromotionsWidget } from '@/components/dashboard/step-promotions-widget';
import { UpcomingChildrenWidget } from '@/components/dashboard/upcoming-children-widget';
import { SectionAgeAlertsWidget } from '@/components/dashboard/section-age-alerts-widget';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { getCurrentMonthRange } from '@/lib/utils/formatting';
import { useAuthStore } from '@/stores/auth-store';

export default function OrgDashboardPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { user } = useAuthStore();

  const { from, to } = getCurrentMonthRange();

  const { data: employeesData, isLoading: employeesLoading } = useQuery({
    queryKey: [...queryKeys.employees.list(orgId, 1), 'count'],
    queryFn: () => apiClient.getEmployees(orgId, { page: 1, limit: 1 }),
    enabled: !!orgId,
    staleTime: 2 * 60 * 1000,
  });

  const { data: childrenData, isLoading: childrenLoading } = useQuery({
    queryKey: [...queryKeys.children.list(orgId, 1), 'count'],
    queryFn: () => apiClient.getChildren(orgId, { page: 1, limit: 1 }),
    enabled: !!orgId,
    staleTime: 2 * 60 * 1000,
  });

  const { data: staffingData, isLoading: staffingLoading } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(orgId, undefined, from, to),
    queryFn: () => apiClient.getStaffingHours(orgId, { from, to }),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000,
  });

  // Check if previous month's bill exists
  const { data: billsData } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.list(orgId, 1),
    queryFn: () => apiClient.getGovernmentFundingBillPeriods(orgId, { page: 1, limit: 100 }),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000,
  });

  const previousMonthMissing = useMemo(() => {
    if (!billsData?.data) return null;
    const now = new Date();
    const prevMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1);
    const prevMonthStr = prevMonth.toISOString().slice(0, 7); // "YYYY-MM"
    const hasBill = billsData.data.some((b) => b.from.startsWith(prevMonthStr));
    if (hasBill) return null;
    return prevMonth.toLocaleDateString('de-DE', { month: 'long', year: 'numeric' });
  }, [billsData]);

  const currentMonth = staffingData?.data_points?.[0];
  const coverageBalance =
    currentMonth && currentMonth.required_hours > 0
      ? Math.round((currentMonth.available_hours / currentMonth.required_hours) * 100) - 100
      : null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('dashboard.title')}</h1>
        <p className="text-muted-foreground">
          {t('dashboard.welcome')}
          {user?.name && `, ${user.name}`}
        </p>
      </div>

      {previousMonthMissing && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-base font-medium">
              {t('governmentFundingBills.missingBillAlert', { month: previousMonthMissing })}
            </CardTitle>
            <Button variant="outline" size="sm" asChild>
              <Link href={`/organizations/${orgId}/government-funding-bills`}>
                <Upload className="mr-1 h-3 w-3" />
                {t('governmentFundingBills.uploadBill')}
              </Link>
            </Button>
          </CardHeader>
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard
          title={t('dashboard.activeEmployees')}
          value={employeesData?.total ?? '-'}
          icon={Users}
          loading={employeesLoading}
        />
        <StatCard
          title={t('dashboard.activeChildren')}
          value={childrenData?.total ?? '-'}
          icon={Baby}
          loading={childrenLoading}
        />
        <StatCard
          title={t('dashboard.staffingCoverage')}
          value={
            coverageBalance !== null ? `${coverageBalance >= 0 ? '+' : ''}${coverageBalance}%` : '-'
          }
          description={
            currentMonth
              ? `${Math.round(currentMonth.available_hours)}h / ${Math.round(currentMonth.required_hours)}h`
              : undefined
          }
          valueClassName={
            coverageBalance !== null
              ? coverageBalance >= 0
                ? 'text-green-600'
                : 'text-red-600'
              : undefined
          }
          icon={Clock}
          loading={staffingLoading}
        />
      </div>

      <StepPromotionsWidget orgId={orgId} />
      <UpcomingChildrenWidget orgId={orgId} />
      <SectionAgeAlertsWidget orgId={orgId} />
    </div>
  );
}
