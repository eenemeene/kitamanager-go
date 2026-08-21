'use client';

import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle, XCircle, Thermometer, Palmtree, CircleDashed } from 'lucide-react';
import { StatCard } from '@/components/dashboard/stat-card';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';

interface AttendanceSummaryProps {
  orgId: number;
  date: string;
}

export function AttendanceSummary({ orgId, date }: AttendanceSummaryProps) {
  const t = useTranslations('attendance');

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.attendance.summary(orgId, date),
    queryFn: () => apiClient.getChildAttendanceSummary(orgId, date),
    enabled: !!orgId,
  });

  const recorded =
    (data?.present ?? 0) + (data?.absent ?? 0) + (data?.sick ?? 0) + (data?.vacation ?? 0);
  const notRecorded = (data?.total_children ?? 0) - recorded;

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
      <StatCard
        title={t('present')}
        value={data?.present ?? 0}
        icon={CheckCircle}
        loading={isLoading}
        valueClassName="text-success"
      />
      <StatCard
        title={t('absent')}
        value={data?.absent ?? 0}
        icon={XCircle}
        loading={isLoading}
        valueClassName="text-destructive"
      />
      <StatCard
        title={t('sick')}
        value={data?.sick ?? 0}
        icon={Thermometer}
        loading={isLoading}
        valueClassName="text-warning"
      />
      <StatCard
        title={t('vacation')}
        value={data?.vacation ?? 0}
        icon={Palmtree}
        loading={isLoading}
        valueClassName="text-info"
      />
      <StatCard
        title={t('notRecorded')}
        value={notRecorded < 0 ? 0 : notRecorded}
        icon={CircleDashed}
        loading={isLoading}
        valueClassName="text-muted-foreground"
      />
    </div>
  );
}
