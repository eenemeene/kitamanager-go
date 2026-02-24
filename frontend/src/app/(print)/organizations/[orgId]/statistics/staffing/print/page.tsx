'use client';

import { useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import { useParams } from 'next/navigation';
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
import { getCurrentMonthStart } from '@/lib/utils/formatting';

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
  const [year] = useState(new Date().getFullYear());

  const from = `${year}-01-01`;
  const to = `${year}-12-01`;

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

  const { data: staffingHours, isLoading: isLoadingStaffing } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(orgId),
    queryFn: () => apiClient.getStaffingHours(orgId),
    enabled: !!orgId,
  });

  const { data: staffingGrid, isLoading: isLoadingGrid } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(orgId, undefined, from, to),
    queryFn: () => apiClient.getStaffingHours(orgId, { from, to }),
    enabled: !!orgId,
  });

  const { data: employeeStaffingGrid, isLoading: isLoadingEmployeeGrid } = useQuery({
    queryKey: queryKeys.statistics.employeeStaffingHours(orgId, undefined, from, to),
    queryFn: () => apiClient.getEmployeeStaffingHours(orgId, { from, to }),
    enabled: !!orgId,
  });

  const sectionStaffingQueries = useQueries({
    queries: (sections?.data ?? []).map((section) => ({
      queryKey: queryKeys.statistics.staffingHours(orgId, section.id),
      queryFn: () => apiClient.getStaffingHours(orgId, { sectionId: section.id }),
      enabled: !!orgId && !!sections,
    })),
  });

  const sectionStaffingData = useMemo(() => {
    if (!sections?.data) return [];
    const currentMonth = getCurrentMonthStart();
    return sections.data
      .map((section, i) => {
        const queryResult = sectionStaffingQueries[i];
        if (!queryResult?.data?.data_points?.length) return null;
        const points = queryResult.data.data_points;
        const exact = points.find((dp) => dp.date === currentMonth);
        const dp = exact ?? points[points.length - 1];
        return {
          sectionName: section.name,
          required: dp.required_hours,
          available: dp.available_hours,
        };
      })
      .filter((d): d is NonNullable<typeof d> => d !== null);
  }, [sections?.data, sectionStaffingQueries]);

  return (
    <div className="mx-auto max-w-[1100px] p-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsStaffing')}</h1>
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

      {/* Staffing Hours Chart */}
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

      {/* Staffing by Section */}
      {sectionStaffingData.length > 0 && (
        <div className="mb-8 break-inside-avoid">
          <h2 className="mb-3 text-xl font-semibold">{t('statistics.sectionStaffing')}</h2>
          <ChartErrorBoundary>
            <SectionStaffingChart data={sectionStaffingData} />
          </ChartErrorBoundary>
        </div>
      )}

      {/* Staffing Hours Grid */}
      <div className="mb-8 break-inside-avoid">
        <h2 className="mb-3 text-xl font-semibold">
          {t('statistics.staffingHoursGrid')} &middot; {year}
        </h2>
        {isLoadingGrid ? (
          <Skeleton className="h-[200px] w-full" />
        ) : staffingGrid ? (
          <ChartErrorBoundary>
            <StaffingHoursTable data={staffingGrid} />
          </ChartErrorBoundary>
        ) : null}
      </div>

      {/* Employee Staffing Hours Grid */}
      <div className="break-inside-avoid">
        <h2 className="mb-3 text-xl font-semibold">
          {t('statistics.employeeStaffingHoursGrid')} &middot; {year}
        </h2>
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
