'use client';

import { useState } from 'react';
import dynamic from 'next/dynamic';
import { useTranslations } from 'next-intl';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { StaffingHoursTable } from '@/components/charts/staffing-hours-table';
import { EmployeeStaffingHoursTable } from '@/components/charts/employee-staffing-hours-table';
import { OccupancyTable } from '@/components/charts/occupancy-table';
import type { ForecastResponse } from '@/lib/api/types';

const FinancialsChart = dynamic(
  () => import('@/components/charts/financials-bar-chart').then((mod) => mod.FinancialsChart),
  { ssr: false, loading: () => <Skeleton className="h-[580px] w-full" /> }
);

const StaffingHoursChart = dynamic(
  () => import('@/components/charts/staffing-hours-chart').then((mod) => mod.StaffingHoursChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
);

interface BaselineData {
  financials?: ForecastResponse['financials'];
  staffingHours?: ForecastResponse['staffing_hours'];
  occupancy?: ForecastResponse['occupancy'];
  employeeStaffingHours?: ForecastResponse['employee_staffing_hours'];
  isLoading: boolean;
}

interface ForecastResultsProps {
  data: ForecastResponse;
  baseline?: BaselineData;
}

export function ForecastResults({ data, baseline }: ForecastResultsProps) {
  const t = useTranslations('statistics');
  const [showBaseline, setShowBaseline] = useState(false);

  const hasBaseline = baseline && !baseline.isLoading;

  return (
    <Card>
      <CardHeader className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <CardTitle>{t('forecastResults')}</CardTitle>
        {baseline && (
          <div className="flex items-center gap-2">
            <Switch id="show-baseline" checked={showBaseline} onCheckedChange={setShowBaseline} />
            <Label htmlFor="show-baseline" className="text-sm font-normal">
              {t('forecastShowBaseline')}
            </Label>
          </div>
        )}
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="financials">
          <TabsList className="flex flex-wrap">
            <TabsTrigger value="financials">{t('forecastTabFinancials')}</TabsTrigger>
            <TabsTrigger value="staffing">{t('forecastTabStaffing')}</TabsTrigger>
            <TabsTrigger value="occupancy">{t('forecastTabOccupancy')}</TabsTrigger>
            <TabsTrigger value="employeeHours">{t('forecastTabEmployeeHours')}</TabsTrigger>
          </TabsList>

          <TabsContent value="financials">
            {showBaseline && hasBaseline ? (
              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <h4 className="text-muted-foreground mb-2 text-sm font-medium">
                    {t('forecastBaseline')}
                  </h4>
                  {baseline.financials ? (
                    <ChartErrorBoundary>
                      <FinancialsChart data={baseline.financials} />
                    </ChartErrorBoundary>
                  ) : (
                    <p className="text-muted-foreground">{t('chartError')}</p>
                  )}
                </div>
                <div>
                  <h4 className="text-muted-foreground mb-2 text-sm font-medium">
                    {t('forecastScenario')}
                  </h4>
                  {data.financials ? (
                    <ChartErrorBoundary>
                      <FinancialsChart data={data.financials} />
                    </ChartErrorBoundary>
                  ) : (
                    <p className="text-muted-foreground">{t('chartError')}</p>
                  )}
                </div>
              </div>
            ) : data.financials ? (
              <ChartErrorBoundary>
                <FinancialsChart data={data.financials} />
              </ChartErrorBoundary>
            ) : (
              <p className="text-muted-foreground">{t('chartError')}</p>
            )}
          </TabsContent>

          <TabsContent value="staffing">
            {showBaseline && hasBaseline ? (
              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div className="space-y-6">
                  <h4 className="text-muted-foreground text-sm font-medium">
                    {t('forecastBaseline')}
                  </h4>
                  {baseline.staffingHours ? (
                    <>
                      <ChartErrorBoundary>
                        <StaffingHoursChart data={baseline.staffingHours} />
                      </ChartErrorBoundary>
                      <ChartErrorBoundary>
                        <StaffingHoursTable data={baseline.staffingHours} />
                      </ChartErrorBoundary>
                    </>
                  ) : (
                    <p className="text-muted-foreground">{t('chartError')}</p>
                  )}
                </div>
                <div className="space-y-6">
                  <h4 className="text-muted-foreground text-sm font-medium">
                    {t('forecastScenario')}
                  </h4>
                  {data.staffing_hours ? (
                    <>
                      <ChartErrorBoundary>
                        <StaffingHoursChart data={data.staffing_hours} />
                      </ChartErrorBoundary>
                      <ChartErrorBoundary>
                        <StaffingHoursTable data={data.staffing_hours} />
                      </ChartErrorBoundary>
                    </>
                  ) : (
                    <p className="text-muted-foreground">{t('chartError')}</p>
                  )}
                </div>
              </div>
            ) : data.staffing_hours ? (
              <div className="space-y-6">
                <ChartErrorBoundary>
                  <StaffingHoursChart data={data.staffing_hours} />
                </ChartErrorBoundary>
                <ChartErrorBoundary>
                  <StaffingHoursTable data={data.staffing_hours} />
                </ChartErrorBoundary>
              </div>
            ) : (
              <p className="text-muted-foreground">{t('chartError')}</p>
            )}
          </TabsContent>

          <TabsContent value="occupancy">
            {showBaseline && hasBaseline ? (
              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <h4 className="text-muted-foreground mb-2 text-sm font-medium">
                    {t('forecastBaseline')}
                  </h4>
                  {baseline.occupancy ? (
                    <ChartErrorBoundary>
                      <OccupancyTable data={baseline.occupancy} />
                    </ChartErrorBoundary>
                  ) : (
                    <p className="text-muted-foreground">{t('chartError')}</p>
                  )}
                </div>
                <div>
                  <h4 className="text-muted-foreground mb-2 text-sm font-medium">
                    {t('forecastScenario')}
                  </h4>
                  {data.occupancy ? (
                    <ChartErrorBoundary>
                      <OccupancyTable data={data.occupancy} />
                    </ChartErrorBoundary>
                  ) : (
                    <p className="text-muted-foreground">{t('chartError')}</p>
                  )}
                </div>
              </div>
            ) : data.occupancy ? (
              <ChartErrorBoundary>
                <OccupancyTable data={data.occupancy} />
              </ChartErrorBoundary>
            ) : (
              <p className="text-muted-foreground">{t('chartError')}</p>
            )}
          </TabsContent>

          <TabsContent value="employeeHours">
            {showBaseline && hasBaseline ? (
              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <h4 className="text-muted-foreground mb-2 text-sm font-medium">
                    {t('forecastBaseline')}
                  </h4>
                  {baseline.employeeStaffingHours ? (
                    <ChartErrorBoundary>
                      <EmployeeStaffingHoursTable data={baseline.employeeStaffingHours} />
                    </ChartErrorBoundary>
                  ) : (
                    <p className="text-muted-foreground">{t('chartError')}</p>
                  )}
                </div>
                <div>
                  <h4 className="text-muted-foreground mb-2 text-sm font-medium">
                    {t('forecastScenario')}
                  </h4>
                  {data.employee_staffing_hours ? (
                    <ChartErrorBoundary>
                      <EmployeeStaffingHoursTable data={data.employee_staffing_hours} />
                    </ChartErrorBoundary>
                  ) : (
                    <p className="text-muted-foreground">{t('chartError')}</p>
                  )}
                </div>
              </div>
            ) : data.employee_staffing_hours ? (
              <ChartErrorBoundary>
                <EmployeeStaffingHoursTable data={data.employee_staffing_hours} />
              </ChartErrorBoundary>
            ) : (
              <p className="text-muted-foreground">{t('chartError')}</p>
            )}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
