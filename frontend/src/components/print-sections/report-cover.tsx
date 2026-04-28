'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { formatCurrency } from '@/lib/utils/formatting';
import { useFrontendVersion } from '@/hooks/use-frontend-version';
import { type ReportMonth, formatReportMonthLong } from '@/lib/utils/report-month';

interface Props {
  orgId: number;
  orgName: string;
  reportMonth: ReportMonth;
}

/**
 * Cover / front page of the combined report. Reads as a real
 * report cover: tool wordmark, prominent title, short intro,
 * headline KPIs, and a generation footnote.
 *
 * Reuses the same query keys the body sections fire so TanStack
 * Query dedupes — no extra round-trips. The cover just picks the
 * report-month data point out of those wider responses.
 */
export function ReportCover({ orgId, orgName, reportMonth }: Props) {
  const t = useTranslations();
  const webVersion = useFrontendVersion();

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

  const generatedAt = new Date().toLocaleDateString();
  const monthLabel = formatReportMonthLong(reportMonth);

  return (
    <section className="report-section break-inside-avoid">
      {/* Top-of-page wordmark + main title */}
      <div className="mb-8">
        <p className="text-muted-foreground text-sm font-semibold tracking-wider uppercase">
          {t('report.title')}
        </p>
        <h1 className="mt-2 text-5xl font-bold tracking-tight">
          {t('report.cover.titleLine', { month: monthLabel, org: orgName })}
        </h1>
      </div>

      {/* Short intro paragraph — gives the reader the "why" before the KPIs */}
      <p className="text-muted-foreground mb-8 max-w-3xl text-base leading-relaxed">
        {t('report.cover.intro', { month: monthLabel, org: orgName })}
      </p>

      {/* Headline KPIs */}
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

      {/* Generation footnote — pinned to bottom of cover via mt-auto.
          The CLI's per-page colophon adds API + CLI versions; this one
          works for browser-print where there is no CLI. */}
      <div className="text-muted-foreground mt-12 border-t pt-4 text-xs">
        {t('report.cover.generatedFootnote', {
          date: generatedAt,
          web: webVersion || 'dev',
        })}
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
