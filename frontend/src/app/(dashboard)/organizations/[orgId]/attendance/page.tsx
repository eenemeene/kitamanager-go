'use client';

import { useMemo, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries, useMutation, useQueryClient } from '@tanstack/react-query';
import { startOfWeek, addDays, eachDayOfInterval } from 'date-fns';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { WeekStepper } from '@/components/ui/week-stepper';
import { AttendanceWeekTable } from '@/components/attendance/attendance-week-table';
import { QueryError } from '@/components/crud/query-error';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { ChildAttendanceResponse, ChildAttendanceStatus, Section } from '@/lib/api/types';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { useToast } from '@/lib/hooks/use-toast';
import { useState } from 'react';

export default function AttendancePage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations('attendance');
  const tStats = useTranslations('statistics');
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [selectedDate, setSelectedDate] = useState(() => new Date());
  const [sectionFilter, setSectionFilter] = useState<number | undefined>(undefined);

  // Compute Mon-Fri dates
  const weekMonday = useMemo(() => startOfWeek(selectedDate, { weekStartsOn: 1 }), [selectedDate]);
  const weekDays = useMemo(
    () =>
      eachDayOfInterval({
        start: weekMonday,
        end: addDays(weekMonday, 4),
      }),
    [weekMonday]
  );
  const weekMondayStr = weekMonday.toISOString().slice(0, 10);

  // Fetch sections for filter dropdown
  const { data: sectionsData } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });
  const sections: Section[] = sectionsData?.data ?? [];

  // Fetch children with active contracts for the week
  const {
    data: weekChildren,
    isLoading: childrenLoading,
    error: childrenError,
    refetch: refetchChildren,
  } = useQuery({
    queryKey: [...queryKeys.children.allUnpaginated(orgId), weekMondayStr, sectionFilter],
    queryFn: () => apiClient.getChildrenAllForDate(orgId, weekMondayStr, sectionFilter),
    enabled: !!orgId,
  });

  // Fetch attendance for all 5 weekdays in parallel
  const weekAttendanceQueries = useQueries({
    queries: weekDays.map((day) => {
      const dayStr = day.toISOString().slice(0, 10);
      return {
        queryKey: queryKeys.attendance.byDate(orgId, dayStr),
        queryFn: () => apiClient.getChildAttendanceByDateAll(orgId, dayStr),
        enabled: !!orgId,
      };
    }),
  });

  const attendanceLoading = weekAttendanceQueries.some((q) => q.isLoading);
  const attendanceError = weekAttendanceQueries.find((q) => q.error)?.error ?? null;

  const weekAttendanceByDate = useMemo(() => {
    const map = new Map<string, ChildAttendanceResponse[]>();
    weekDays.forEach((day, i) => {
      const dayStr = day.toISOString().slice(0, 10);
      map.set(dayStr, weekAttendanceQueries[i]?.data ?? []);
    });
    return map;
  }, [weekDays, weekAttendanceQueries]);

  const isLoading = childrenLoading || attendanceLoading;
  const queryError = childrenError || attendanceError;

  const invalidateDate = useCallback(
    (dateStr: string) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.attendance.byDate(orgId, dateStr) });
    },
    [queryClient, orgId]
  );

  // Check-in mutation: create attendance with status=present and check_in_time=now
  const checkInMutation = useMutation({
    mutationFn: async ({ childId, forDate }: { childId: number; forDate: string }) => {
      const now = new Date();
      const checkInTime = `${forDate}T${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:00Z`;
      return apiClient.createChildAttendance(orgId, childId, {
        date: forDate,
        status: 'present',
        check_in_time: checkInTime,
      });
    },
    onSuccess: (_data, variables) => {
      invalidateDate(variables.forDate);
    },
    onError: () => {
      toast({ title: t('failedToSave'), variant: 'destructive' });
    },
  });

  // Check-out mutation: update attendance with check_out_time=now
  const checkOutMutation = useMutation({
    mutationFn: async ({
      childId,
      forDate,
      attendanceId,
    }: {
      childId: number;
      forDate: string;
      attendanceId: number;
    }) => {
      const now = new Date();
      const checkOutTime = `${forDate}T${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:00Z`;
      return apiClient.updateChildAttendance(orgId, childId, attendanceId, {
        check_out_time: checkOutTime,
      });
    },
    onSuccess: (_data, variables) => {
      invalidateDate(variables.forDate);
    },
    onError: () => {
      toast({ title: t('failedToSave'), variant: 'destructive' });
    },
  });

  const handleCheckIn = useCallback(
    (childId: number, forDate: string) => {
      checkInMutation.mutate({ childId, forDate });
    },
    [checkInMutation]
  );

  const handleCheckOut = useCallback(
    (childId: number, forDate: string, attendanceId: number) => {
      checkOutMutation.mutate({ childId, forDate, attendanceId });
    },
    [checkOutMutation]
  );

  // Update time mutation: edit check_in_time or check_out_time
  const updateTimeMutation = useMutation({
    mutationFn: async ({
      childId,
      forDate,
      attendanceId,
      field,
      time,
    }: {
      childId: number;
      forDate: string;
      attendanceId: number;
      field: 'check_in_time' | 'check_out_time';
      time: string;
    }) => {
      const isoTime = `${forDate}T${time}:00Z`;
      return apiClient.updateChildAttendance(orgId, childId, attendanceId, {
        [field]: isoTime,
      });
    },
    onSuccess: (_data, variables) => {
      invalidateDate(variables.forDate);
      toast({ title: t('updateSuccess') });
    },
    onError: () => {
      toast({ title: t('failedToSave'), variant: 'destructive' });
    },
  });

  const handleUpdateTime = useCallback(
    (
      childId: number,
      forDate: string,
      attendanceId: number,
      field: 'check_in_time' | 'check_out_time',
      time: string
    ) => {
      updateTimeMutation.mutate({ childId, forDate, attendanceId, field, time });
    },
    [updateTimeMutation]
  );

  // Set status mutation: create or update status (absent, sick, vacation, present)
  const setStatusMutation = useMutation({
    mutationFn: async ({
      childId,
      forDate,
      status,
      attendanceId,
    }: {
      childId: number;
      forDate: string;
      status: ChildAttendanceStatus;
      attendanceId?: number;
    }) => {
      if (attendanceId) {
        return apiClient.updateChildAttendance(orgId, childId, attendanceId, { status });
      }
      return apiClient.createChildAttendance(orgId, childId, { date: forDate, status });
    },
    onSuccess: (_data, variables) => {
      invalidateDate(variables.forDate);
      toast({ title: t('updateSuccess') });
    },
    onError: () => {
      toast({ title: t('failedToSave'), variant: 'destructive' });
    },
  });

  const handleSetStatus = useCallback(
    (childId: number, forDate: string, status: ChildAttendanceStatus, attendanceId?: number) => {
      setStatusMutation.mutate({ childId, forDate, status, attendanceId });
    },
    [setStatusMutation]
  );

  // Save note mutation
  const saveNoteMutation = useMutation({
    mutationFn: async ({
      childId,
      attendanceId,
      note,
    }: {
      childId: number;
      forDate: string;
      attendanceId: number;
      note: string;
    }) => {
      return apiClient.updateChildAttendance(orgId, childId, attendanceId, {
        note: note || undefined,
      });
    },
    onSuccess: (_data, variables) => {
      invalidateDate(variables.forDate);
      toast({ title: t('updateSuccess') });
    },
    onError: () => {
      toast({ title: t('failedToSave'), variant: 'destructive' });
    },
  });

  const handleSaveNote = useCallback(
    (childId: number, forDate: string, attendanceId: number, note: string) => {
      saveNoteMutation.mutate({ childId, forDate, attendanceId, note });
    },
    [saveNoteMutation]
  );

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold tracking-tight">{t('title')}</h1>

      <div className="flex flex-wrap items-center gap-4">
        <WeekStepper value={selectedDate} onChange={setSelectedDate} />
        <Select
          value={sectionFilter ? String(sectionFilter) : 'all'}
          onValueChange={(value) => setSectionFilter(value === 'all' ? undefined : Number(value))}
        >
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder={tStats('filterBySection')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{tStats('allSections')}</SelectItem>
            {sections.map((section) => (
              <SelectItem key={section.id} value={String(section.id)}>
                {section.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('title')}</CardTitle>
        </CardHeader>
        <CardContent>
          {queryError ? (
            <QueryError
              error={queryError}
              onRetry={() => {
                refetchChildren();
                weekAttendanceQueries.forEach((q) => q.refetch());
              }}
            />
          ) : isLoading ? (
            <div className="space-y-2">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <AttendanceWeekTable
              childRecords={weekChildren ?? []}
              attendanceByDate={weekAttendanceByDate}
              onCheckIn={handleCheckIn}
              onCheckOut={handleCheckOut}
              onUpdateTime={handleUpdateTime}
              onSetStatus={handleSetStatus}
              onSaveNote={handleSaveNote}
              days={weekDays}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
