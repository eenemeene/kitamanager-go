'use client';

import { useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useParams, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { StaffingHoursTable } from '@/components/charts/staffing-hours-table';
import { EmployeeStaffingHoursTable } from '@/components/charts/employee-staffing-hours-table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { useUiStore } from '@/stores/ui-store';
import {
  parseReportMonth,
  formatReportMonthLong,
  formatKitaYearLabel,
} from '@/lib/utils/report-month';

const StaffingHoursChart = dynamic(
  () => import('@/components/charts/staffing-hours-chart').then((mod) => mod.StaffingHoursChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

const SectionStaffingChart = dynamic(
  () =>
    import('@/components/charts/section-staffing-chart').then((mod) => mod.SectionStaffingChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

export default function StaffingPrintPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { organizations, fetchOrganizations } = useUiStore();
  const searchParams = useSearchParams();
  const reportMonth = useMemo(() => parseReportMonth(searchParams.get('month')), [searchParams]);

  // Ensure organizations are loaded for org name
  useQuery({
    queryKey: ['organizations-load'],
    queryFn: async () => {
      if (organizations.length === 0) await fetchOrganizations();
      return null;
    },
  });

  const orgName = organizations.find((o) => o.id === orgId)?.name ?? '';

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  // Multi-year trend chart: previous + current + next Kita year.
  const { data: staffingHours, isLoading: isLoadingStaffing } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(
      orgId,
      undefined,
      reportMonth.trendFrom,
      reportMonth.trendTo
    ),
    queryFn: () =>
      apiClient.getStaffingHours(orgId, {
        from: reportMonth.trendFrom,
        to: reportMonth.trendTo,
      }),
    enabled: !!orgId,
  });

  // Annual grids: Kita year (Aug → Jul) containing the report month.
  const { data: staffingGrid, isLoading: isLoadingGrid } = useQuery({
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

  const { data: employeeStaffingGrid, isLoading: isLoadingEmployeeGrid } = useQuery({
    queryKey: queryKeys.statistics.employeeStaffingHours(
      orgId,
      undefined,
      reportMonth.kitaYearFrom,
      reportMonth.kitaYearTo
    ),
    queryFn: () =>
      apiClient.getEmployeeStaffingHours(orgId, {
        from: reportMonth.kitaYearFrom,
        to: reportMonth.kitaYearTo,
      }),
    enabled: !!orgId,
  });

  // Per-section staffing — used only to pick the report-month data point
  // for the SectionStaffingChart's "this month, by section" view. Fetch the
  // Kita year so the picker always has a row available even if the report
  // month is at the very start of a Kita year.
  const sectionStaffingQueries = useQueries({
    queries: (sections?.data ?? []).map((section) => ({
      queryKey: queryKeys.statistics.staffingHours(
        orgId,
        section.id,
        reportMonth.kitaYearFrom,
        reportMonth.kitaYearTo
      ),
      queryFn: () =>
        apiClient.getStaffingHours(orgId, {
          sectionId: section.id,
          from: reportMonth.kitaYearFrom,
          to: reportMonth.kitaYearTo,
        }),
      enabled: !!orgId && !!sections,
    })),
  });

  const sectionStaffingData = useMemo(() => {
    if (!sections?.data) return [];
    return sections.data
      .map((section, i) => {
        const queryResult = sectionStaffingQueries[i];
        if (!queryResult?.data?.data_points?.length) return null;
        const points = queryResult.data.data_points;
        // Pick the report-month data point. Falls back to the latest
        // available point if for some reason the report month isn't in
        // the response (shouldn't happen given the Kita-year window).
        const exact = points.find((dp) => dp.date === reportMonth.asOf);
        const dp = exact ?? points[points.length - 1];
        return {
          sectionName: section.name,
          required: dp.required_hours,
          available: dp.available_hours,
        };
      })
      .filter((d): d is NonNullable<typeof d> => d !== null);
  }, [sections?.data, sectionStaffingQueries, reportMonth.asOf]);

  return (
    <div
      className="mx-auto max-w-[1100px] p-8"
      data-print-ready={
        !isLoadingStaffing && !isLoadingGrid && !isLoadingEmployeeGrid ? 'true' : undefined
      }
    >
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {t('nav.statisticsStaffing')} &middot; {formatReportMonthLong(reportMonth)}
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

      {/* Staffing Hours Chart — multi-year trend */}
      <div className="mb-8 break-inside-avoid">
        <h2 className="mb-3 text-xl font-semibold">{t('statistics.staffingHours')}</h2>
        {isLoadingStaffing ? (
          <Skeleton className="h-[300px] w-full" />
        ) : staffingHours ? (
          <ChartErrorBoundary>
            <StaffingHoursChart data={staffingHours} />
          </ChartErrorBoundary>
        ) : null}
      </div>

      {/* Staffing by Section — snapshot at the report month */}
      {sectionStaffingData.length > 0 && (
        <div className="mb-8 break-inside-avoid">
          <h2 className="mb-3 text-xl font-semibold">{t('statistics.sectionStaffing')}</h2>
          <ChartErrorBoundary>
            <SectionStaffingChart data={sectionStaffingData} />
          </ChartErrorBoundary>
        </div>
      )}

      {/* Staffing Hours Grid — Kita year */}
      <div className="mb-8 break-inside-avoid">
        <h2 className="mb-1 text-xl font-semibold">{t('statistics.staffingHoursGrid')}</h2>
        <p className="text-muted-foreground mb-3 text-sm">{formatKitaYearLabel(reportMonth)}</p>
        {isLoadingGrid ? (
          <Skeleton className="h-[200px] w-full" />
        ) : staffingGrid ? (
          <ChartErrorBoundary>
            <StaffingHoursTable data={staffingGrid} />
          </ChartErrorBoundary>
        ) : null}
      </div>

      {/* Employee Staffing Hours Grid — Kita year */}
      <div className="break-inside-avoid">
        <h2 className="mb-1 text-xl font-semibold">{t('statistics.employeeStaffingHoursGrid')}</h2>
        <p className="text-muted-foreground mb-3 text-sm">{formatKitaYearLabel(reportMonth)}</p>
        {isLoadingEmployeeGrid ? (
          <Skeleton className="h-[200px] w-full" />
        ) : employeeStaffingGrid ? (
          <ChartErrorBoundary>
            <EmployeeStaffingHoursTable data={employeeStaffingGrid} />
          </ChartErrorBoundary>
        ) : null}
      </div>
    </div>
  );
}
