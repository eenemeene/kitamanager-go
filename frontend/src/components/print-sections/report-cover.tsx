'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { formatCurrency } from '@/lib/utils/formatting';
import { type ReportMonth, formatReportMonthLong } from '@/lib/utils/report-month';

interface Props {
  orgId: number;
  orgName: string;
  reportMonth: ReportMonth;
}

/**
 * Cover / executive-summary section for the combined report page.
 * Headline KPIs at a glance for the report month: head-counts, the
 * month's coverage, and the month's bottom line.
 *
 * Reuses the same query keys the body sections fire so TanStack
 * Query dedupes — no extra round-trips. The cover just picks the
 * report-month data point out of those wider responses.
 */
export function ReportCover({ orgId, orgName, reportMonth }: Props) {
  const t = useTranslations();

  const { data: childrenList } = useQuery({
    queryKey: [...queryKeys.children.list(orgId, 1), 'count'],
    queryFn: () => apiClient.getChildren(orgId, { page: 1, limit: 1 }),
    enabled: !!orgId,
  });

  const { data: employeesList } = useQuery({
    queryKey: [...queryKeys.employees.list(orgId, 1), 'count'],
    queryFn: () => apiClient.getEmployees(orgId, { page: 1, limit: 1 }),
    enabled: !!orgId,
  });

  // Same key as the staffing section's annual grid — cache hit.
  const { data: staffingGrid } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(
      orgId,
      undefined,
      reportMonth.kitaYearFrom,
      reportMonth.kitaYearTo
    ),
    queryFn: () =>
      apiClient.getStaffingHours(orgId, {
        from: reportMonth.kitaYearFrom,
        to: reportMonth.kitaYearTo,
      }),
    enabled: !!orgId,
  });

  // Same key as the financials section's trend chart — cache hit.
  const { data: financials } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId, reportMonth.trendFrom, reportMonth.trendTo),
    queryFn: () =>
      apiClient.getFinancials(orgId, {
        from: reportMonth.trendFrom,
        to: reportMonth.trendTo,
      }),
    enabled: !!orgId,
  });

  const reportMonthStaffing = useMemo(
    () => staffingGrid?.data_points?.find((dp) => dp.date === reportMonth.asOf),
    [staffingGrid, reportMonth.asOf]
  );

  const reportMonthFinancials = useMemo(
    () => financials?.data_points?.find((dp) => dp.date === reportMonth.asOf),
    [financials, reportMonth.asOf]
  );

  const coverage =
    reportMonthStaffing && reportMonthStaffing.required_hours > 0
      ? Math.round(
          (reportMonthStaffing.available_hours / reportMonthStaffing.required_hours) * 100
        ) - 100
      : null;

  return (
    <section className="report-section mb-8 break-inside-avoid">
      <div className="mb-6 border-b pb-6">
        <h1 className="text-4xl font-bold tracking-tight">{orgName}</h1>
        <p className="text-muted-foreground mt-1 text-lg">
          {t('report.cover.subtitle', { month: formatReportMonthLong(reportMonth) })}
        </p>
      </div>

      <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
        <KpiCard
          label={t('dashboard.activeChildren')}
          value={childrenList?.total != null ? String(childrenList.total) : '—'}
        />
        <KpiCard
          label={t('dashboard.activeEmployees')}
          value={employeesList?.total != null ? String(employeesList.total) : '—'}
        />
        <KpiCard
          label={t('dashboard.staffingCoverage')}
          value={coverage != null ? `${coverage >= 0 ? '+' : ''}${coverage}%` : '—'}
          tone={coverage != null ? (coverage >= 0 ? 'success' : 'destructive') : undefined}
        />
        <KpiCard
          label={t('statistics.totalIncome')}
          value={reportMonthFinancials ? formatCurrency(reportMonthFinancials.total_income) : '—'}
          tone="success"
        />
        <KpiCard
          label={t('statistics.totalExpenses')}
          value={reportMonthFinancials ? formatCurrency(reportMonthFinancials.total_expenses) : '—'}
          tone="destructive"
        />
        <KpiCard
          label={t('statistics.balance')}
          value={reportMonthFinancials ? formatCurrency(reportMonthFinancials.balance) : '—'}
          tone={
            reportMonthFinancials
              ? reportMonthFinancials.balance >= 0
                ? 'info'
                : 'destructive'
              : undefined
          }
        />
      </div>
    </section>
  );
}

function KpiCard({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: 'success' | 'destructive' | 'info';
}) {
  const toneClass =
    tone === 'success'
      ? 'text-success'
      : tone === 'destructive'
        ? 'text-destructive'
        : tone === 'info'
          ? 'text-info'
          : '';
  return (
    <div className="rounded-lg border p-4">
      <p className="text-muted-foreground text-sm font-medium">{label}</p>
      <p className={`mt-1 text-2xl font-bold ${toneClass}`.trim()}>{value}</p>
    </div>
  );
}
