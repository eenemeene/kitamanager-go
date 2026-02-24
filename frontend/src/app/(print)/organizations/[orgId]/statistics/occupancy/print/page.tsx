'use client';

import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { ChartErrorBoundary } from '@/components/charts/chart-error-boundary';
import { OccupancyTable } from '@/components/charts/occupancy-table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { useUiStore } from '@/stores/ui-store';

export default function OccupancyPrintPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { organizations, fetchOrganizations } = useUiStore();

  useQuery({
    queryKey: ['organizations-load'],
    queryFn: async () => {
      if (organizations.length === 0) await fetchOrganizations();
      return null;
    },
  });

  const orgName = organizations.find((o) => o.id === orgId)?.name ?? '';

  const { data: occupancy, isLoading } = useQuery({
    queryKey: queryKeys.statistics.occupancy(orgId),
    queryFn: () => apiClient.getOccupancy(orgId),
    enabled: !!orgId,
  });

  return (
    <div className="mx-auto max-w-[1100px] p-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('nav.statisticsOccupancy')}</h1>
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

      <div className="break-inside-avoid">
        <h2 className="mb-3 text-xl font-semibold">{t('statistics.occupancyMatrix')}</h2>
        {isLoading ? (
          <Skeleton className="h-[300px] w-full" />
        ) : occupancy ? (
          <ChartErrorBoundary>
            <OccupancyTable data={occupancy} />
          </ChartErrorBoundary>
        ) : null}
      </div>
    </div>
  );
}
