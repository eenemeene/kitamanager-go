'use client';

import { useMemo, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries, useMutation, useQueryClient } from '@tanstack/react-query';
import { startOfWeek, addDays, eachDayOfInterval, format } from 'date-fns';
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
import { ToastAction } from '@/components/ui/toast';
import { useState } from 'react';

export default function AttendancePage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations('attendance');
  const tStats = useTranslations('statistics');
  const tCommon = useTranslations('common');
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
  const weekMondayStr = format(weekMonday, 'yyyy-MM-dd');

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
      const dayStr = format(day, 'yyyy-MM-dd');
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
      const dayStr = format(day, 'yyyy-MM-dd');
      map.set(dayStr, weekAttendanceQueries[i]?.data ?? []);
    });
    return map;
  }, [weekDays, weekAttendanceQueries]);

  const isLoading = childrenLoading || attendanceLoading;
  const queryError = childrenError || attendanceError;

  const invalidateAttendance = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.attendance.all(orgId) });
  }, [queryClient, orgId]);

  const getChildName = useCallback(
    (childId: number): string => {
      const child = weekChildren?.find((c) => c.id === childId);
      return child ? `${child.first_name} ${child.last_name}` : '';
    },
    [weekChildren]
  );

  // Check-in mutation: create attendance with status=present and check_in_time=now
  const checkInMutation = useMutation({
    mutationFn: async ({ childId, forDate }: { childId: number; forDate: string }) => {
      const checkInTime = new Date().toISOString();
      return apiClient.createChildAttendance(orgId, childId, {
        date: forDate,
        status: 'present',
        check_in_time: checkInTime,
      });
    },
    onSuccess: (data, variables) => {
      invalidateAttendance();
      toast({
        title: t('checkedIn', { name: getChildName(variables.childId) }),
        action: (
          <ToastAction
            altText={t('undo')}
            onClick={() => {
              apiClient
                .deleteChildAttendance(orgId, variables.childId, data.id)
                .then(() => {
                  invalidateAttendance();
                  toast({ title: t('undone') });
                })
                .catch(() => {
                  toast({ title: t('undoFailed'), variant: 'destructive' });
                });
            }}
          >
            {t('undo')}
          </ToastAction>
        ),
      });
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
      const checkOutTime = new Date().toISOString();
      return apiClient.updateChildAttendance(orgId, childId, attendanceId, {
        check_out_time: checkOutTime,
      });
    },
    onSuccess: (_data, variables) => {
      invalidateAttendance();
      toast({
        title: t('checkedOut', { name: getChildName(variables.childId) }),
        action: (
          <ToastAction
            altText={t('undo')}
            onClick={() => {
              apiClient
                .updateChildAttendance(orgId, variables.childId, variables.attendanceId, {
                  // null, not '': the field is a Go time.Time, which cannot
                  // parse an empty string and answered 400. An explicit null is
                  // what says "clear it" -- omitting the field means "leave it
                  // alone", so there is no third spelling that works here.
                  check_out_time: null,
                })
                .then(() => {
                  invalidateAttendance();
                  toast({ title: t('undone') });
                })
                .catch(() => {
                  toast({ title: t('undoFailed'), variant: 'destructive' });
                });
            }}
          >
            {t('undo')}
          </ToastAction>
        ),
      });
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
      const localDate = new Date(`${forDate}T${time}:00`);
      const isoTime = localDate.toISOString();
      return apiClient.updateChildAttendance(orgId, childId, attendanceId, {
        [field]: isoTime,
      });
    },
    onSuccess: (_data, variables) => {
      invalidateAttendance();
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
      previousStatus,
    }: {
      childId: number;
      forDate: string;
      status: ChildAttendanceStatus;
      attendanceId?: number;
      previousStatus?: ChildAttendanceStatus;
    }) => {
      if (attendanceId) {
        return apiClient.updateChildAttendance(orgId, childId, attendanceId, { status });
      }
      return apiClient.createChildAttendance(orgId, childId, { date: forDate, status });
    },
    onSuccess: (data, variables) => {
      invalidateAttendance();
      const statusLabel = t(variables.status);
      toast({
        title: t('statusChanged', {
          name: getChildName(variables.childId),
          status: statusLabel,
        }),
        action: (
          <ToastAction
            altText={t('undo')}
            onClick={() => {
              const undoPromise = variables.attendanceId
                ? // Was an update — revert to previous status
                  apiClient.updateChildAttendance(orgId, variables.childId, data.id, {
                    status: variables.previousStatus!,
                  })
                : // Was a create — delete the record
                  apiClient.deleteChildAttendance(orgId, variables.childId, data.id);

              undoPromise
                .then(() => {
                  invalidateAttendance();
                  toast({ title: t('undone') });
                })
                .catch(() => {
                  toast({ title: t('undoFailed'), variant: 'destructive' });
                });
            }}
          >
            {t('undo')}
          </ToastAction>
        ),
      });
    },
    onError: () => {
      toast({ title: t('failedToSave'), variant: 'destructive' });
    },
  });

  const handleSetStatus = useCallback(
    (childId: number, forDate: string, status: ChildAttendanceStatus, attendanceId?: number) => {
      // Capture previous status for undo
      let previousStatus: ChildAttendanceStatus | undefined;
      if (attendanceId) {
        const dayRecords = weekAttendanceByDate.get(forDate);
        const existing = dayRecords?.find((r) => r.id === attendanceId);
        previousStatus = existing?.status as ChildAttendanceStatus | undefined;
      }
      setStatusMutation.mutate({ childId, forDate, status, attendanceId, previousStatus });
    },
    [setStatusMutation, weekAttendanceByDate]
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
      invalidateAttendance();
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
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('title')}</h1>
        <p className="text-muted-foreground mt-1 max-w-3xl text-sm">{t('description')}</p>
      </div>

      <div className="flex flex-wrap items-center gap-2 md:gap-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-muted-foreground text-sm">{tCommon('week')}</span>
          <WeekStepper value={selectedDate} onChange={setSelectedDate} />
        </div>
        <Select
          value={sectionFilter ? String(sectionFilter) : 'all'}
          onValueChange={(value) => setSectionFilter(value === 'all' ? undefined : Number(value))}
        >
          <SelectTrigger className="w-full md:w-[200px]">
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
