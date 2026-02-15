'use client';

import { useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';

const AgeDistributionChart = dynamic(
  () =>
    import('@/components/charts/age-distribution-chart').then((mod) => mod.AgeDistributionChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

const MonthlyContractChart = dynamic(
  () =>
    import('@/components/charts/monthly-contract-chart').then((mod) => mod.MonthlyContractChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

const StaffingHoursChart = dynamic(
  () => import('@/components/charts/staffing-hours-chart').then((mod) => mod.StaffingHoursChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

const SectionStaffingChart = dynamic(
  () =>
    import('@/components/charts/section-staffing-chart').then((mod) => mod.SectionStaffingChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

const ContractPropertiesChart = dynamic(
  () =>
    import('@/components/charts/contract-properties-chart').then(
      (mod) => mod.ContractPropertiesChart
    ),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

export default function StatisticsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const [selectedSectionId, setSelectedSectionId] = useState<number | undefined>(undefined);

  const { data: ageDistribution, isLoading: isLoadingAge } = useQuery({
    queryKey: queryKeys.statistics.ageDistribution(orgId),
    queryFn: () => apiClient.getAgeDistribution(orgId),
    enabled: !!orgId,
  });

  const { data: contractCounts, isLoading: isLoadingContracts } = useQuery({
    queryKey: queryKeys.statistics.contractCounts(orgId),
    queryFn: () => apiClient.getChildrenContractCountByMonth(orgId),
    enabled: !!orgId,
  });

  const { data: contractProperties, isLoading: isLoadingContractProperties } = useQuery({
    queryKey: queryKeys.statistics.contractProperties(orgId),
    queryFn: () => apiClient.getContractPropertiesDistribution(orgId),
    enabled: !!orgId,
  });

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: 100 }),
    enabled: !!orgId,
  });

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
    const now = new Date();
    const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`;

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

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('statistics.title')}</h1>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Age Distribution */}
        <Card>
          <CardHeader>
            <CardTitle>{t('statistics.ageDistribution')}</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoadingAge ? (
              <Skeleton className="h-[300px] w-full" />
            ) : ageDistribution ? (
              <AgeDistributionChart data={ageDistribution} />
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>

        {/* Monthly Contract Counts */}
        <Card>
          <CardHeader>
            <CardTitle>{t('statistics.childrenContractCount')}</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoadingContracts ? (
              <Skeleton className="h-[300px] w-full" />
            ) : contractCounts ? (
              <MonthlyContractChart data={contractCounts} />
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>

        {/* Contract Properties Distribution */}
        <Card>
          <CardHeader>
            <CardTitle>{t('statistics.contractProperties')}</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoadingContractProperties ? (
              <Skeleton className="h-[300px] w-full" />
            ) : contractProperties ? (
              <ContractPropertiesChart data={contractProperties} />
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>

        {/* Staffing by Section */}
        {sections && sections.data.length > 0 && (
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>{t('statistics.sectionStaffing')}</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoadingSectionStaffing ? (
                <Skeleton className="h-[300px] w-full" />
              ) : sectionStaffingData.length > 0 ? (
                <SectionStaffingChart data={sectionStaffingData} />
              ) : (
                <p className="text-muted-foreground">{t('statistics.chartError')}</p>
              )}
            </CardContent>
          </Card>
        )}

        {/* Staffing Hours */}
        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <div>
              <CardTitle>{t('statistics.staffingHours')}</CardTitle>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('statistics.staffingHoursDescription')}
              </p>
            </div>
            {sections && sections.data.length > 0 && (
              <Select
                value={selectedSectionId?.toString() ?? 'all'}
                onValueChange={(value) =>
                  setSelectedSectionId(value === 'all' ? undefined : Number(value))
                }
              >
                <SelectTrigger className="w-[200px]">
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
            )}
          </CardHeader>
          <CardContent>
            {isLoadingStaffing ? (
              <Skeleton className="h-[300px] w-full" />
            ) : staffingHours ? (
              <StaffingHoursChart data={staffingHours} />
            ) : (
              <p className="text-muted-foreground">{t('statistics.chartError')}</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
