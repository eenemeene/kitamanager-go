'use client';

import { useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { CrudFormDialog } from '@/components/crud/crud-form-dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { attendanceSchema, type AttendanceFormData } from '@/lib/schemas/attendance';
import type { ChildAttendanceResponse, ChildAttendanceStatus } from '@/lib/api/types';
import { formatTime } from '@/lib/utils/formatting';

interface AttendanceEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  attendance: ChildAttendanceResponse | null;
  childName: string;
  isSaving: boolean;
  onSubmit: (data: AttendanceFormData) => void;
}

const STATUS_OPTIONS: ChildAttendanceStatus[] = ['present', 'absent', 'sick', 'vacation'];

export function AttendanceEditDialog({
  open,
  onOpenChange,
  attendance,
  childName,
  isSaving,
  onSubmit,
}: AttendanceEditDialogProps) {
  const t = useTranslations('attendance');
  const tCommon = useTranslations('common');

  const { register, handleSubmit, reset, setValue, watch } = useForm<AttendanceFormData>({
    resolver: zodResolver(attendanceSchema),
    defaultValues: {
      status: 'present',
      check_in_time: '',
      check_out_time: '',
      note: '',
    },
  });

  useEffect(() => {
    if (open && attendance) {
      reset({
        status: attendance.status,
        check_in_time: formatTime(attendance.check_in_time),
        check_out_time: formatTime(attendance.check_out_time),
        note: attendance.note ?? '',
      });
    }
  }, [open, attendance, reset]);

  const statusValue = watch('status');

  return (
    <CrudFormDialog
      open={open}
      onOpenChange={onOpenChange}
      isEditing={true}
      translationPrefix="attendance"
      onSubmit={handleSubmit(onSubmit)}
      isSaving={isSaving}
    >
      <p className="text-muted-foreground text-sm">{childName}</p>

      <div className="space-y-2">
        <Label>{tCommon('status')}</Label>
        <Select
          value={statusValue}
          onValueChange={(val) => setValue('status', val as ChildAttendanceStatus)}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {STATUS_OPTIONS.map((s) => (
              <SelectItem key={s} value={s}>
                {t(s)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="check_in_time">{t('checkIn')}</Label>
          <Input id="check_in_time" type="time" {...register('check_in_time')} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="check_out_time">{t('checkOut')}</Label>
          <Input id="check_out_time" type="time" {...register('check_out_time')} />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="note">{t('note')}</Label>
        <textarea
          id="note"
          {...register('note')}
          rows={2}
          className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        />
      </div>
    </CrudFormDialog>
  );
}
