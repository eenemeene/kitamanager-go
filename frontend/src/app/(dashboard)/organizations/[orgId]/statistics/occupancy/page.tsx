'use client';

import { useState } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { SectionFilter } from '@/components/ui/section-filter';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { StatisticsPageHeader } from '@/components/statistics/statistics-page-header';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { OccupancyTable } from '@/components/charts/occupancy-table';

/** A YYYY-MM-DD search parameter, or undefined if it is absent or malformed. */
function dateParam(raw: string | null): string | undefined {
  if (raw === null) return undefined;
  return /^\d{4}-\d{2}-\d{2}$/.test(raw) ? raw : undefined;
}

export default function OccupancyPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const searchParams = useSearchParams();
  const [selectedSectionId, setSelectedSectionId] = useState<number | undefined>(undefined);

  // The window is normally the server's default — twelve months back, six
  // ahead — which moves with the calendar. from/to override it, using the same
  // names the endpoint already takes, so a particular window can be linked to
  // and reloaded. Omitting them keeps the previous behaviour exactly.
  //
  // There is no control for this on screen yet; the parameters exist so the
  // window can be addressed at all. That is what lets a test compare this
  // matrix, whose every column is a month and every cell a count for one.
  const from = dateParam(searchParams.get('from'));
  const to = dateParam(searchParams.get('to'));

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const { data: occupancy, isLoading } = useQuery({
    queryKey: queryKeys.statistics.occupancy(orgId, selectedSectionId, from, to),
    queryFn: () => apiClient.getOccupancy(orgId, { sectionId: selectedSectionId, from, to }),
    enabled: !!orgId,
  });

  return (
    <div className="space-y-6">
      <StatisticsPageHeader
        titleKey="nav.statisticsOccupancy"
        descriptionKey="statistics.navOccupancyDescription"
        printHref={`/organizations/${orgId}/statistics/occupancy/print`}
      />

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div>
            <CardTitle>{t('statistics.occupancyMatrix')}</CardTitle>
            <p className="text-muted-foreground mt-1 text-sm">
              {t('statistics.occupancyDescription')}
            </p>
          </div>
          {sections && sections.data.length > 0 && (
            <SectionFilter
              sections={sections.data}
              value={selectedSectionId}
              onChange={setSelectedSectionId}
            />
          )}
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Skeleton className="h-[300px] w-full" />
          ) : occupancy ? (
            <ChartErrorBoundary>
              <OccupancyTable data={occupancy} />
            </ChartErrorBoundary>
          ) : (
            <p className="text-muted-foreground">{t('statistics.chartError')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
