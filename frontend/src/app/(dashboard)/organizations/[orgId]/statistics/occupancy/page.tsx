'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { OccupancyTable } from '@/components/charts/occupancy-table';

export default function OccupancyPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const [selectedSectionId, setSelectedSectionId] = useState<number | undefined>(undefined);

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const { data: occupancy, isLoading } = useQuery({
    queryKey: queryKeys.statistics.occupancy(orgId, selectedSectionId),
    queryFn: () => apiClient.getOccupancy(orgId, { sectionId: selectedSectionId }),
    enabled: !!orgId,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsOccupancy')}</h1>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div>
            <CardTitle>{t('statistics.occupancyMatrix')}</CardTitle>
            <p className="text-muted-foreground mt-1 text-sm">
              {t('statistics.occupancyDescription')}
            </p>
          </div>
          {sections && sections.data.length > 0 && (
            <Select
              value={selectedSectionId?.toString() ?? 'all'}
              onValueChange={(value) =>
                setSelectedSectionId(value === 'all' ? undefined : Number(value))
              }
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
