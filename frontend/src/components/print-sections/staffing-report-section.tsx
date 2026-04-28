'use client';

import { useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries } from '@tanstack/react-query';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { StaffingHoursTable } from '@/components/charts/staffing-hours-table';
import { EmployeeStaffingHoursTable } from '@/components/charts/employee-staffing-hours-table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { type ReportMonth, formatKitaYearLabel } from '@/lib/utils/report-month';

const StaffingHoursChart = dynamic(
  () => import('@/components/charts/staffing-hours-chart').then((mod) => mod.StaffingHoursChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

const SectionStaffingChart = dynamic(
  () =>
    import('@/components/charts/section-staffing-chart').then((mod) => mod.SectionStaffingChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

interface Props {
  orgId: number;
  reportMonth: ReportMonth;
}

/**
 * Staffing section of a print report. Four visualizations:
 *   - Staffing hours timeline (multi-year trend)
 *   - Per-section staffing snapshot (the report month)
 *   - Annual staffing hours grid (Kita year)
 *   - Annual employee staffing hours grid (Kita year)
 */
export function StaffingReportSection({ orgId, reportMonth }: Props) {
  const t = useTranslations();

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

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
    <section className="report-section">
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

      {sectionStaffingData.length > 0 && (
        <div className="mb-8 break-inside-avoid">
          <h2 className="mb-3 text-xl font-semibold">{t('statistics.sectionStaffing')}</h2>
          <ChartErrorBoundary>
            <SectionStaffingChart data={sectionStaffingData} />
          </ChartErrorBoundary>
        </div>
      )}

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
    </section>
  );
}
