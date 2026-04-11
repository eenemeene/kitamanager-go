'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Calculator, RotateCcw } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { SectionFilter } from '@/components/ui/section-filter';
import { YearStepper } from '@/components/ui/year-stepper';
import { ForecastResults } from '@/components/forecast/forecast-results';
import { ForecastModificationSummary } from '@/components/forecast/forecast-modification-summary';
import { ForecastChildrenTab } from '@/components/forecast/forecast-children-tab';
import { ForecastEmployeesTab } from '@/components/forecast/forecast-employees-tab';
import { ForecastSalaryTab } from '@/components/forecast/forecast-salary-tab';
import { ForecastFundingTab } from '@/components/forecast/forecast-funding-tab';
import { ForecastBudgetTab } from '@/components/forecast/forecast-budget-tab';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { useForecastStore } from '@/stores/forecast-store';

export default function ForecastPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const store = useForecastStore();

  const [year, setYear] = useState(new Date().getFullYear());
  const from = `${year}-01-01`;
  const to = `${year}-12-01`;

  // Sync date range to store when year changes
  useEffect(() => {
    store.setFilters(from, to, store.sectionId);
  }, [from, to]); // eslint-disable-line react-hooks/exhaustive-deps

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const handleSectionChange = (sectionId: number | undefined) => {
    store.setFilters(from, to, sectionId);
  };

  // Baseline data queries (fetched alongside forecast for comparison)
  const { data: baselineFinancials, isLoading: isLoadingBaselineFinancials } = useQuery({
    queryKey: queryKeys.statistics.financials(orgId, from, to),
    queryFn: () => apiClient.getFinancials(orgId, { from, to }),
    enabled: !!orgId,
  });

  const { data: baselineStaffing, isLoading: isLoadingBaselineStaffing } = useQuery({
    queryKey: queryKeys.statistics.staffingHours(orgId, store.sectionId, from, to),
    queryFn: () => apiClient.getStaffingHours(orgId, { sectionId: store.sectionId, from, to }),
    enabled: !!orgId,
  });

  const { data: baselineOccupancy, isLoading: isLoadingBaselineOccupancy } = useQuery({
    queryKey: queryKeys.statistics.occupancy(orgId, store.sectionId, from, to),
    queryFn: () => apiClient.getOccupancy(orgId, { sectionId: store.sectionId, from, to }),
    enabled: !!orgId,
  });

  const { data: baselineEmployeeHours, isLoading: isLoadingBaselineEmployeeHours } = useQuery({
    queryKey: queryKeys.statistics.employeeStaffingHours(orgId, store.sectionId, from, to),
    queryFn: () =>
      apiClient.getEmployeeStaffingHours(orgId, { sectionId: store.sectionId, from, to }),
    enabled: !!orgId,
  });

  const baselineLoading =
    isLoadingBaselineFinancials ||
    isLoadingBaselineStaffing ||
    isLoadingBaselineOccupancy ||
    isLoadingBaselineEmployeeHours;

  const forecastMutation = useMutation({
    mutationFn: (req: Parameters<typeof apiClient.postForecast>[1]) =>
      apiClient.postForecast(orgId, req),
  });

  const handleCalculate = () => {
    forecastMutation.mutate(store.buildRequest());
  };

  const handleReset = () => {
    store.reset();
    forecastMutation.reset();
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('statistics.forecastTitle')}</h1>
        <p className="text-muted-foreground mt-1 text-sm">{t('statistics.forecastDescription')}</p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-2 md:gap-4">
        <YearStepper value={year} onChange={setYear} />
        {sections && sections.data.length > 0 && (
          <SectionFilter
            sections={sections.data}
            value={store.sectionId}
            onChange={handleSectionChange}
          />
        )}
      </div>

      {/* Scenario Configuration */}
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.forecastConfigTitle')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Tabs defaultValue="children">
            <TabsList className="flex flex-wrap">
              <TabsTrigger value="children">{t('statistics.forecastTabChildren')}</TabsTrigger>
              <TabsTrigger value="employees">{t('statistics.forecastTabEmployees')}</TabsTrigger>
              <TabsTrigger value="salary">{t('statistics.forecastTabSalary')}</TabsTrigger>
              <TabsTrigger value="funding">{t('statistics.forecastTabFunding')}</TabsTrigger>
              <TabsTrigger value="budget">{t('statistics.forecastTabBudget')}</TabsTrigger>
            </TabsList>

            <TabsContent value="children">
              <ForecastChildrenTab />
            </TabsContent>
            <TabsContent value="employees">
              <ForecastEmployeesTab />
            </TabsContent>
            <TabsContent value="salary">
              <ForecastSalaryTab />
            </TabsContent>
            <TabsContent value="funding">
              <ForecastFundingTab />
            </TabsContent>
            <TabsContent value="budget">
              <ForecastBudgetTab />
            </TabsContent>
          </Tabs>

          {/* Modification Summary */}
          <ForecastModificationSummary />

          {/* Action Buttons */}
          <div className="flex flex-wrap gap-2">
            <Button onClick={handleCalculate} disabled={forecastMutation.isPending}>
              <Calculator className="mr-2 h-4 w-4" />
              {forecastMutation.isPending ? t('common.loading') : t('statistics.forecastCalculate')}
            </Button>
            <Button variant="outline" onClick={handleReset}>
              <RotateCcw className="mr-2 h-4 w-4" />
              {t('statistics.forecastReset')}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Results */}
      {forecastMutation.isPending && <Skeleton className="h-[400px] w-full" />}

      {forecastMutation.isError && (
        <Card>
          <CardContent className="pt-6">
            <p className="text-destructive">
              {t('common.error')}: {forecastMutation.error.message}
            </p>
          </CardContent>
        </Card>
      )}

      {forecastMutation.isSuccess && forecastMutation.data && (
        <ForecastResults
          data={forecastMutation.data}
          baseline={{
            financials: baselineFinancials,
            staffingHours: baselineStaffing,
            occupancy: baselineOccupancy,
            employeeStaffingHours: baselineEmployeeHours,
            isLoading: baselineLoading,
          }}
        />
      )}

      {!forecastMutation.isPending && !forecastMutation.isSuccess && !forecastMutation.isError && (
        <Card>
          <CardContent className="pt-6">
            <p className="text-muted-foreground text-sm">{t('statistics.forecastNoResults')}</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
