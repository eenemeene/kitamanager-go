'use client';

import { useMemo } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Users, Baby, Clock, Upload, ArrowRight, Check } from 'lucide-react';
import { StatCard } from '@/components/dashboard/stat-card';
import { StepPromotionsWidget } from '@/components/dashboard/step-promotions-widget';
import { UpcomingChildrenWidget } from '@/components/dashboard/upcoming-children-widget';
import { SectionAgeAlertsWidget } from '@/components/dashboard/section-age-alerts-widget';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { getCurrentMonthRange } from '@/lib/utils/formatting';
import { useAuthStore } from '@/stores/auth-store';
import { useUiStore } from '@/stores/ui-store';

export default function OrgDashboardPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { user } = useAuthStore();
  const queryClient = useQueryClient();

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

  // Accept suggestion: rename child to match bill name and assign the voucher
  const acceptSuggestion = useMutation({
    mutationFn: async ({
      childId,
      firstName,
      lastName,
      voucherNumber,
    }: {
      childId: number;
      firstName: string;
      lastName: string;
      voucherNumber: string;
    }) => {
      await apiClient.updateChild(orgId, childId, {
        first_name: firstName,
        last_name: lastName,
      });
      await apiClient.assignChildVoucher(orgId, childId, voucherNumber);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
    },
  });

  // Property mismatches (latest bill vs contracts)
  const { data: latestCompareWrapped } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.compareLatest(orgId),
    queryFn: () => apiClient.compareBills(orgId),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000,
  });
  const latestCompare = latestCompareWrapped?.comparisons?.[0];

  const propertyMismatches = useMemo(() => {
    if (!latestCompare?.children) return [];
    return latestCompare.children.filter((child) => child.properties?.some((p) => !!p.mismatch));
  }, [latestCompare]);

  const currentMonth = staffingData?.data_points?.[0];
  const requiredHours = currentMonth?.required_hours ?? 0;
  const availableHours = currentMonth?.available_hours ?? 0;
  const coverageBalance =
    currentMonth && requiredHours > 0
      ? Math.round((availableHours / requiredHours) * 100) - 100
      : null;

  const orgName = useUiStore((state) => state.organizations.find((o) => o.id === orgId)?.name);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          {orgName ? t('dashboard.titleForOrg', { name: orgName }) : t('dashboard.title')}
        </h1>
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
          tooltip={t('dashboard.activeEmployeesTooltip')}
          value={employeesData?.total ?? '-'}
          icon={Users}
          loading={employeesLoading}
        />
        <StatCard
          title={t('dashboard.activeChildren')}
          tooltip={t('dashboard.activeChildrenTooltip')}
          value={childrenData?.total ?? '-'}
          icon={Baby}
          loading={childrenLoading}
        />
        <StatCard
          title={t('dashboard.staffingCoverage')}
          tooltip={t('dashboard.staffingCoverageTooltip')}
          value={
            coverageBalance !== null ? `${coverageBalance >= 0 ? '+' : ''}${coverageBalance}%` : '-'
          }
          description={
            currentMonth
              ? `${Math.round(availableHours)}h / ${Math.round(requiredHours)}h`
              : undefined
          }
          valueClassName={
            coverageBalance !== null
              ? coverageBalance >= 0
                ? 'text-success'
                : 'text-destructive'
              : undefined
          }
          icon={Clock}
          loading={staffingLoading}
        />
      </div>

      {childrenWithoutVouchers && childrenWithoutVouchers.length > 0 && (
        <Card>
          <CardHeader className="space-y-1 pb-2">
            <div className="flex flex-row items-center justify-between">
              <CardTitle className="text-base font-medium">
                {t('dashboard.childrenWithoutVouchers')}
              </CardTitle>
              <Badge variant="secondary">{childrenWithoutVouchers.length}</Badge>
            </div>
            <CardDescription>{t('dashboard.childrenWithoutVouchersDescription')}</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableBody>
                {childrenWithoutVouchers.map((child) => (
                  <TableRow key={child.id}>
                    <TableCell>
                      <div>
                        <Link
                          href={`/organizations/${orgId}/children/${child.id}`}
                          className="hover:text-primary hover:underline"
                        >
                          {child.first_name} {child.last_name}
                        </Link>
                        {child.suggestions && child.suggestions.length > 0 && (
                          <div className="mt-1 space-y-1">
                            {child.suggestions.map((s) => (
                              <div
                                key={s.voucher_number}
                                className="text-muted-foreground flex items-center gap-2 text-sm"
                              >
                                <ArrowRight className="h-3 w-3" />
                                <span>
                                  {s.bill_first_name} {s.bill_last_name}{' '}
                                  <span className="text-xs">
                                    ({Math.round((s.similarity ?? 0) * 100)}%)
                                  </span>
                                </span>
                                <Tooltip delayDuration={0}>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      className="h-7 px-2 text-xs"
                                      disabled={acceptSuggestion.isPending}
                                      onClick={() =>
                                        acceptSuggestion.mutate({
                                          childId: child.id,
                                          firstName: s.bill_first_name ?? '',
                                          lastName: s.bill_last_name ?? '',
                                          voucherNumber: s.voucher_number ?? '',
                                        })
                                      }
                                    >
                                      <Check className="mr-1 h-3 w-3" />
                                      {t('dashboard.acceptSuggestion')}
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent className="max-w-xs">
                                    {t('dashboard.acceptSuggestionTooltip')}
                                  </TooltipContent>
                                </Tooltip>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
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
          <CardHeader className="space-y-1 pb-2">
            <div className="flex flex-row items-center justify-between">
              <CardTitle className="text-base font-medium">
                {t('dashboard.propertyMismatches')}
              </CardTitle>
              <Badge variant="destructive">{propertyMismatches.length}</Badge>
            </div>
            <CardDescription>{t('dashboard.propertyMismatchesDescription')}</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('children.title')}</TableHead>
                  <TableHead>{t('dashboard.mismatchProperty')}</TableHead>
                  <TableHead>{t('dashboard.mismatchBillValue')}</TableHead>
                  <TableHead>{t('dashboard.mismatchContractValue')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {propertyMismatches.flatMap((child) => {
                  // Group mismatched properties by key so a "different" pair
                  // (one row from the bill, one from the contract) collapses
                  // to a single visual row with bill and contract values
                  // shown side-by-side.
                  const mismatched = child.properties?.filter((p) => !!p.mismatch) ?? [];
                  const byKey = new Map<string, typeof mismatched>();
                  for (const p of mismatched) {
                    const key = p.key ?? '';
                    const group = byKey.get(key) ?? [];
                    group.push(p);
                    byKey.set(key, group);
                  }

                  const rows: Array<{
                    propLabel: string;
                    billCell: React.ReactNode;
                    contractCell: React.ReactNode;
                  }> = [];

                  for (const [key, props] of byKey) {
                    if (props.length >= 2 && props[0].mismatch === 'different') {
                      const bill = props.find((p) => p.bill_amount !== null);
                      const calc = props.find((p) => p.calculated_amount !== null);
                      rows.push({
                        propLabel: bill?.label || calc?.label || key,
                        billCell: bill?.value ?? '?',
                        contractCell: calc?.value ?? '?',
                      });
                    } else {
                      for (const p of props) {
                        const label = p.label || `${p.key}: ${p.value}`;
                        if (p.mismatch === 'missing') {
                          rows.push({
                            propLabel: label,
                            billCell: <span className="text-muted-foreground">—</span>,
                            contractCell: p.value ?? '?',
                          });
                        } else if (p.mismatch === 'additional') {
                          rows.push({
                            propLabel: label,
                            billCell: p.value ?? '?',
                            contractCell: <span className="text-muted-foreground">—</span>,
                          });
                        }
                      }
                    }
                  }

                  const childKey = child.child_id ?? child.voucher_number ?? '';
                  return rows.map((r, idx) => (
                    <TableRow key={`${childKey}-${idx}`}>
                      <TableCell>
                        {idx === 0 ? (
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
                        ) : null}
                      </TableCell>
                      <TableCell className="text-sm">{r.propLabel}</TableCell>
                      <TableCell className="text-sm">{r.billCell}</TableCell>
                      <TableCell className="text-sm">{r.contractCell}</TableCell>
                    </TableRow>
                  ));
                })}
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
