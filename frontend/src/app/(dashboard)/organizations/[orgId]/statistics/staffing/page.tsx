'use client';

import { useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { StaffingHoursTable } from '@/components/charts/staffing-hours-table';
import { EmployeeStaffingHoursTable } from '@/components/charts/employee-staffing-hours-table';
import { YearStepper } from '@/components/ui/year-stepper';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
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

export default function StaffingPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const [selectedSectionId, setSelectedSectionId] = useState<number | undefined>(undefined);
  const [year, setYear] = useState(new Date().getFullYear());

  const from = `${year}-01-01`;
  const to = `${year}-12-01`;

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  // Grid data: year-scoped query
  const { data: staffingGrid, isLoading: isLoadingGrid } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(orgId, selectedSectionId, from, to),
    queryFn: () => apiClient.getStaffingHours(orgId, { sectionId: selectedSectionId, from, to }),
    enabled: !!orgId,
  });

  // Employee staffing hours grid: year-scoped query
  const { data: employeeStaffingGrid, isLoading: isLoadingEmployeeGrid } = useQuery({
    queryKey: queryKeys.statistics.employeeStaffingHours(orgId, selectedSectionId, from, to),
    queryFn: () =>
      apiClient.getEmployeeStaffingHours(orgId, { sectionId: selectedSectionId, from, to }),
    enabled: !!orgId,
  });

  // Chart data: default (no date range)
  const { data: staffingHours, isLoading: isLoadingStaffing } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(orgId, selectedSectionId),
    queryFn: () => apiClient.getStaffingHours(orgId, { sectionId: selectedSectionId }),
    enabled: !!orgId,
  });

  // Fetch staffing hours per section for the grouped bar chart
  const sectionStaffingQueries = useQueries({
    queries: (sections?.data ?? []).map((section) => ({
      queryKey: queryKeys.statistics.staffingHours(orgId, section.id),
      queryFn: () => apiClient.getStaffingHours(orgId, { sectionId: section.id }),
      enabled: !!orgId && !!sections,
    })),
  });

  const sectionStaffingData = useMemo(() => {
    if (!sections?.data) return [];

    // Find the data point closest to the 1st of the current month
    const currentMonth = getCurrentMonthStart();

    return sections.data
      .map((section, i) => {
        const queryResult = sectionStaffingQueries[i];
        if (!queryResult?.data?.data_points?.length) return null;

        // Find the data point for the current month, or the closest one
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

  const isLoadingSectionStaffing =
    sectionStaffingQueries.length > 0 && sectionStaffingQueries.some((q) => q.isLoading);

  const sectionFilter = sections && sections.data.length > 0 && (
    <Select
      value={selectedSectionId?.toString() ?? 'all'}
      onValueChange={(value) => setSelectedSectionId(value === 'all' ? undefined : Number(value))}
    >
      <SelectTrigger className="w-full md:w-[200px]">
        <SelectValue placeholder={t('statistics.filterBySection')} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="all">{t('statistics.allSections')}</SelectItem>
        {sections.data.map((section) => (
          <SelectItem key={section.id} value={section.id.toString()}>
            {section.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsStaffing')}</h1>
        <Link
          href={`/organizations/${orgId}/statistics/staffing/print`}
          target="_blank"
          className="text-muted-foreground hover:text-foreground inline-flex h-9 w-9 items-center justify-center rounded-md transition-colors"
          title={t('common.print')}
        >
          <Printer className="h-4 w-4" />
        </Link>
      </div>

      {/* Staffing Hours Chart */}
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.staffingHours')}</CardTitle>
          <p className="text-muted-foreground mt-1 text-sm">
            {t('statistics.staffingHoursDescription')}
          </p>
        </CardHeader>
        <CardContent>
          {isLoadingStaffing ? (
            <Skeleton className="h-[300px] w-full" />
          ) : staffingHours ? (
            <ChartErrorBoundary>
              <StaffingHoursChart data={staffingHours} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>

      {/* Staffing by Section */}
      {sections && sections.data.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>{t('statistics.sectionStaffing')}</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoadingSectionStaffing ? (
              <Skeleton className="h-[300px] w-full" />
            ) : sectionStaffingData.length > 0 ? (
              <ChartErrorBoundary>
                <SectionStaffingChart data={sectionStaffingData} />
              </ChartErrorBoundary>
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Staffing Hours Grid */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div>
            <CardTitle>{t('statistics.staffingHoursGrid')}</CardTitle>
          </div>
          <div className="flex flex-wrap items-center gap-2 md:gap-4">
            {sectionFilter}
            <YearStepper value={year} onChange={setYear} />
          </div>
        </CardHeader>
        <CardContent>
          {isLoadingGrid ? (
            <Skeleton className="h-[200px] w-full" />
          ) : staffingGrid ? (
            <ChartErrorBoundary>
              <StaffingHoursTable data={staffingGrid} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>

      {/* Employee Staffing Hours Grid */}
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.employeeStaffingHoursGrid')}</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoadingEmployeeGrid ? (
            <Skeleton className="h-[200px] w-full" />
          ) : employeeStaffingGrid ? (
            <ChartErrorBoundary>
              <EmployeeStaffingHoursTable data={employeeStaffingGrid} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
