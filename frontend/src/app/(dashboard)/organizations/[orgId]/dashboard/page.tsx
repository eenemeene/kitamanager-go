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
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCell, TableRow } from '@/components/ui/table';
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
    staleTime: 60 * 1000, // 1 minute — shorter so bill uploads/deletes are reflected quickly
  });

  const previousMonthMissing = useMemo(() => {
    if (!billsData?.data) return null;
    const now = new Date();
    const prevMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1);
    // Use local date components — toISOString() converts to UTC which shifts the month in non-UTC timezones
    const prevMonthStr = `${prevMonth.getFullYear()}-${String(prevMonth.getMonth() + 1).padStart(2, '0')}`;
    const hasBill = billsData.data.some((b) => b.from.startsWith(prevMonthStr));
    if (hasBill) return null;
    return prevMonth.toLocaleDateString('de-DE', { month: 'long', year: 'numeric' });
  }, [billsData]);

  // Children without vouchers
  const { data: childrenWithoutVouchers } = useQuery({
    queryKey: queryKeys.children.withoutVouchers(orgId),
    queryFn: () => apiClient.getChildrenWithoutVouchers(orgId),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000,
  });

  // Property mismatches (latest bill vs contracts)
  const { data: latestCompare } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.compareLatest(orgId),
    queryFn: () => apiClient.compareBills(orgId),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000,
  });

  const propertyMismatches = useMemo(() => {
    if (!latestCompare?.children) return [];
    return latestCompare.children.filter((child) => child.properties?.some((p) => !!p.mismatch));
  }, [latestCompare]);

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

      {childrenWithoutVouchers && childrenWithoutVouchers.length > 0 && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-base font-medium">
              {t('dashboard.childrenWithoutVouchers')}
            </CardTitle>
            <Badge variant="secondary">{childrenWithoutVouchers.length}</Badge>
          </CardHeader>
          <CardContent>
            <Table>
              <TableBody>
                {childrenWithoutVouchers.map((child) => (
                  <TableRow key={child.id}>
                    <TableCell>
                      <Link
                        href={`/organizations/${orgId}/children/${child.id}/billing`}
                        className="hover:text-primary hover:underline"
                      >
                        {child.first_name} {child.last_name}
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {propertyMismatches.length > 0 && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-base font-medium">
              {t('dashboard.propertyMismatches')}
            </CardTitle>
            <Badge variant="destructive">{propertyMismatches.length}</Badge>
          </CardHeader>
          <CardContent>
            <Table>
              <TableBody>
                {propertyMismatches.map((child) => (
                  <TableRow key={child.child_id ?? child.voucher_number}>
                    <TableCell>
                      <Link
                        href={
                          child.child_id
                            ? `/organizations/${orgId}/children/${child.child_id}/billing`
                            : `/organizations/${orgId}/government-funding-bills`
                        }
                        className="hover:text-primary hover:underline"
                      >
                        {child.child_name}
                      </Link>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {(() => {
                        const mismatched = child.properties?.filter((p) => !!p.mismatch) ?? [];
                        // Group "different" mismatches by key to show bill vs contract
                        const byKey = new Map<string, typeof mismatched>();
                        for (const p of mismatched) {
                          const group = byKey.get(p.key) ?? [];
                          group.push(p);
                          byKey.set(p.key, group);
                        }
                        return Array.from(byKey.entries())
                          .map(([key, props]) => {
                            if (props.length >= 2 && props[0].mismatch === 'different') {
                              const billVal = props.find((p) => p.bill_amount !== null)?.value;
                              const calcVal = props.find(
                                (p) => p.calculated_amount !== null
                              )?.value;
                              return `${key}: ${billVal ?? '?'} (${t('dashboard.billValue')}) / ${calcVal ?? '?'} (${t('dashboard.contractValue')})`;
                            }
                            return props
                              .map((p) => {
                                const label = p.label || `${p.key}: ${p.value}`;
                                const mm = p.mismatch!;
                                return `${label} (${t(`dashboard.mismatch${mm.charAt(0).toUpperCase() + mm.slice(1)}` as Parameters<typeof t>[0])})`;
                              })
                              .join(', ');
                          })
                          .join('; ');
                      })()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <StepPromotionsWidget orgId={orgId} />
      <UpcomingChildrenWidget orgId={orgId} />
      <SectionAgeAlertsWidget orgId={orgId} />
    </div>
  );
}
