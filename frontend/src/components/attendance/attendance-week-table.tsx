'use client';

import { useState, useRef, useEffect } from 'react';
import { format } from 'date-fns';
import { de, enUS } from 'date-fns/locale';
import { useLocale, useTranslations } from 'next-intl';
import {
  LogIn,
  LogOut,
  MoreHorizontal,
  CheckCircle,
  XCircle,
  Thermometer,
  Palmtree,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { ChildAttendanceResponse, ChildAttendanceStatus } from '@/lib/api/types';
import type { Child } from '@/lib/api/types';
import { formatTime } from '@/lib/utils/formatting';

const dateFnsLocales: Record<string, typeof de> = {
  de: de,
  en: enUS,
};

const STATUS_BUTTONS: {
  status: ChildAttendanceStatus;
  icon: typeof CheckCircle;
  color: string;
  activeColor: string;
}[] = [
  {
    status: 'present',
    icon: CheckCircle,
    color: 'text-green-600 hover:bg-green-50',
    activeColor: 'bg-green-100 text-green-700 border-green-300',
  },
  {
    status: 'absent',
    icon: XCircle,
    color: 'text-red-600 hover:bg-red-50',
    activeColor: 'bg-red-100 text-red-700 border-red-300',
  },
  {
    status: 'sick',
    icon: Thermometer,
    color: 'text-orange-600 hover:bg-orange-50',
    activeColor: 'bg-orange-100 text-orange-700 border-orange-300',
  },
  {
    status: 'vacation',
    icon: Palmtree,
    color: 'text-blue-600 hover:bg-blue-50',
    activeColor: 'bg-blue-100 text-blue-700 border-blue-300',
  },
];

const STATUS_ICON_MAP: Record<ChildAttendanceStatus, { icon: typeof CheckCircle; color: string }> =
  {
    present: { icon: CheckCircle, color: 'text-green-600' },
    absent: { icon: XCircle, color: 'text-red-600' },
    sick: { icon: Thermometer, color: 'text-orange-600' },
    vacation: { icon: Palmtree, color: 'text-blue-600' },
  };

// --- EditableTime ---

interface EditableTimeProps {
  value: string;
  className?: string;
  onSave: (newTime: string) => void;
  ariaLabel: string;
}

function EditableTime({ value, className, onSave, ariaLabel }: EditableTimeProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [editing]);

  const handleSave = () => {
    setEditing(false);
    if (draft && draft !== value) {
      onSave(draft);
    } else {
      setDraft(value);
    }
  };

  if (editing) {
    return (
      <input
        ref={inputRef}
        type="time"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={handleSave}
        onKeyDown={(e) => {
          if (e.key === 'Enter') handleSave();
          if (e.key === 'Escape') {
            setDraft(value);
            setEditing(false);
          }
        }}
        className="border-primary h-7 w-[5.5rem] rounded border px-1 text-center text-sm"
        aria-label={ariaLabel}
      />
    );
  }

  return (
    <button
      type="button"
      className={`hover:bg-muted cursor-pointer rounded px-1 text-sm font-medium underline decoration-dotted underline-offset-2 ${className ?? ''}`}
      onClick={() => {
        setDraft(value);
        setEditing(true);
      }}
      aria-label={ariaLabel}
    >
      {value}
    </button>
  );
}

// --- EditableNote ---

interface EditableNoteProps {
  value: string;
  onSave: (newNote: string) => void;
}

function EditableNote({ value, onSave }: EditableNoteProps) {
  const t = useTranslations('attendance');
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (editing && textareaRef.current) {
      textareaRef.current.focus();
      // Place cursor at end
      textareaRef.current.selectionStart = textareaRef.current.value.length;
    }
  }, [editing]);

  const handleSave = () => {
    setEditing(false);
    const trimmed = draft.trim();
    if (trimmed !== value) {
      onSave(trimmed);
    }
  };

  if (editing) {
    return (
      <textarea
        ref={textareaRef}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={handleSave}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            setDraft(value);
            setEditing(false);
          }
        }}
        rows={2}
        className="border-primary mt-0.5 w-full rounded border px-1 py-0.5 text-[0.7rem] leading-tight"
        placeholder={t('note')}
        aria-label={t('note')}
      />
    );
  }

  return (
    <button
      type="button"
      className="text-muted-foreground hover:text-foreground mt-0.5 max-w-[8rem] cursor-pointer truncate text-left text-[0.65rem] leading-tight underline decoration-dotted underline-offset-2"
      onClick={() => {
        setDraft(value);
        setEditing(true);
      }}
      title={value}
      aria-label={t('note')}
    >
      {value}
    </button>
  );
}

// --- StatusNotePopover ---

interface StatusNotePopoverProps {
  attendance: ChildAttendanceResponse | undefined;
  childId: number;
  dateStr: string;
  onSetStatus: (
    childId: number,
    dateStr: string,
    status: ChildAttendanceStatus,
    attendanceId?: number
  ) => void;
  onSaveNote: (childId: number, dateStr: string, attendanceId: number, note: string) => void;
}

function StatusNotePopover({
  attendance,
  childId,
  dateStr,
  onSetStatus,
  onSaveNote,
}: StatusNotePopoverProps) {
  const t = useTranslations('attendance');
  const tCommon = useTranslations('common');
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState(attendance?.note ?? '');

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
    if (nextOpen) {
      setNote(attendance?.note ?? '');
    }
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0"
          aria-label={t('quickMark')}
        >
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-56 space-y-3 p-3" align="center">
        <div className="flex gap-1">
          {STATUS_BUTTONS.map(({ status, icon: Icon, color, activeColor }) => {
            const isActive = attendance?.status === status;
            return (
              <Button
                key={status}
                variant="outline"
                size="icon"
                className={`h-8 w-8 ${isActive ? activeColor : color}`}
                onClick={() => {
                  onSetStatus(childId, dateStr, status, attendance?.id);
                  setOpen(false);
                }}
                aria-label={t(status)}
              >
                <Icon className="h-4 w-4" />
              </Button>
            );
          })}
        </div>
        {attendance && (
          <div className="space-y-1.5">
            <textarea
              className="border-input placeholder:text-muted-foreground w-full rounded-md border px-2 py-1.5 text-sm"
              rows={2}
              placeholder={t('note')}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
            <Button
              size="sm"
              className="w-full"
              onClick={() => {
                onSaveNote(childId, dateStr, attendance.id, note);
                setOpen(false);
              }}
            >
              {tCommon('save')}
            </Button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

// --- AttendanceCell ---

interface AttendanceCellProps {
  attendance: ChildAttendanceResponse | undefined;
  childId: number;
  dateStr: string;
  onCheckIn: (childId: number, dateStr: string) => void;
  onCheckOut: (childId: number, dateStr: string, attendanceId: number) => void;
  onUpdateTime: (
    childId: number,
    dateStr: string,
    attendanceId: number,
    field: 'check_in_time' | 'check_out_time',
    time: string
  ) => void;
  onSetStatus: (
    childId: number,
    dateStr: string,
    status: ChildAttendanceStatus,
    attendanceId?: number
  ) => void;
  onSaveNote: (childId: number, dateStr: string, attendanceId: number, note: string) => void;
}

function AttendanceCell({
  attendance,
  childId,
  dateStr,
  onCheckIn,
  onCheckOut,
  onUpdateTime,
  onSetStatus,
  onSaveNote,
}: AttendanceCellProps) {
  const t = useTranslations('attendance');

  const noteSnippet = attendance?.note ? (
    <EditableNote
      value={attendance.note}
      onSave={(newNote) => onSaveNote(childId, dateStr, attendance.id, newNote)}
    />
  ) : null;

  // No record — show check-in button + more options
  if (!attendance) {
    return (
      <TooltipProvider>
        <div className="flex items-center gap-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-8 gap-1 text-green-600 hover:bg-green-50 hover:text-green-700"
                onClick={() => onCheckIn(childId, dateStr)}
                aria-label={t('checkIn')}
              >
                <LogIn className="h-4 w-4" />
                <span className="hidden sm:inline">{t('checkIn')}</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t('checkIn')}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <StatusNotePopover
                  attendance={attendance}
                  childId={childId}
                  dateStr={dateStr}
                  onSetStatus={onSetStatus}
                  onSaveNote={onSaveNote}
                />
              </span>
            </TooltipTrigger>
            <TooltipContent>{t('quickMark')}</TooltipContent>
          </Tooltip>
        </div>
      </TooltipProvider>
    );
  }

  const checkIn = formatTime(attendance.check_in_time);
  const checkOut = formatTime(attendance.check_out_time);

  // Checked in but not checked out
  if (checkIn && !checkOut) {
    return (
      <TooltipProvider>
        <div>
          <div className="flex items-center gap-1">
            <EditableTime
              value={checkIn}
              className="text-green-700"
              onSave={(newTime) =>
                onUpdateTime(childId, dateStr, attendance.id, 'check_in_time', newTime)
              }
              ariaLabel={t('checkIn')}
            />
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 gap-1 text-orange-600 hover:bg-orange-50 hover:text-orange-700"
                  onClick={() => onCheckOut(childId, dateStr, attendance.id)}
                  aria-label={t('checkOut')}
                >
                  <LogOut className="h-4 w-4" />
                  <span className="hidden sm:inline">{t('checkOut')}</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t('checkOut')}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <StatusNotePopover
                    attendance={attendance}
                    childId={childId}
                    dateStr={dateStr}
                    onSetStatus={onSetStatus}
                    onSaveNote={onSaveNote}
                  />
                </span>
              </TooltipTrigger>
              <TooltipContent>{t('quickMark')}</TooltipContent>
            </Tooltip>
          </div>
          {noteSnippet}
        </div>
      </TooltipProvider>
    );
  }

  // Checked out — both times editable
  if (checkIn && checkOut) {
    return (
      <TooltipProvider>
        <div>
          <div className="flex items-center gap-1">
            <EditableTime
              value={checkIn}
              className="text-green-700"
              onSave={(newTime) =>
                onUpdateTime(childId, dateStr, attendance.id, 'check_in_time', newTime)
              }
              ariaLabel={t('checkIn')}
            />
            <span className="text-muted-foreground text-sm">–</span>
            <EditableTime
              value={checkOut}
              className="text-muted-foreground"
              onSave={(newTime) =>
                onUpdateTime(childId, dateStr, attendance.id, 'check_out_time', newTime)
              }
              ariaLabel={t('checkOut')}
            />
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <StatusNotePopover
                    attendance={attendance}
                    childId={childId}
                    dateStr={dateStr}
                    onSetStatus={onSetStatus}
                    onSaveNote={onSaveNote}
                  />
                </span>
              </TooltipTrigger>
              <TooltipContent>{t('quickMark')}</TooltipContent>
            </Tooltip>
          </div>
          {noteSnippet}
        </div>
      </TooltipProvider>
    );
  }

  // Has record but no times (status-only, e.g. marked absent/sick via popover)
  const StatusIcon = STATUS_ICON_MAP[attendance.status]?.icon;
  const statusColor = STATUS_ICON_MAP[attendance.status]?.color ?? '';
  return (
    <TooltipProvider>
      <div>
        <div className="flex items-center gap-1">
          {StatusIcon && <StatusIcon className={`h-3.5 w-3.5 ${statusColor}`} />}
          <span className={`text-sm italic ${statusColor}`}>{t(attendance.status)}</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <StatusNotePopover
                  attendance={attendance}
                  childId={childId}
                  dateStr={dateStr}
                  onSetStatus={onSetStatus}
                  onSaveNote={onSaveNote}
                />
              </span>
            </TooltipTrigger>
            <TooltipContent>{t('quickMark')}</TooltipContent>
          </Tooltip>
        </div>
        {noteSnippet}
      </div>
    </TooltipProvider>
  );
}

// --- AttendanceWeekTable ---

interface AttendanceWeekTableProps {
  childRecords: Child[];
  attendanceByDate: Map<string, ChildAttendanceResponse[]>;
  onCheckIn: (childId: number, dateStr: string) => void;
  onCheckOut: (childId: number, dateStr: string, attendanceId: number) => void;
  onUpdateTime: (
    childId: number,
    dateStr: string,
    attendanceId: number,
    field: 'check_in_time' | 'check_out_time',
    time: string
  ) => void;
  onSetStatus: (
    childId: number,
    dateStr: string,
    status: ChildAttendanceStatus,
    attendanceId?: number
  ) => void;
  onSaveNote: (childId: number, dateStr: string, attendanceId: number, note: string) => void;
  days: Date[];
}

export function AttendanceWeekTable({
  childRecords,
  attendanceByDate,
  onCheckIn,
  onCheckOut,
  onUpdateTime,
  onSetStatus,
  onSaveNote,
  days,
}: AttendanceWeekTableProps) {
  const t = useTranslations('attendance');
  const tCommon = useTranslations('common');
  const locale = useLocale();
  const dfLocale = dateFnsLocales[locale] ?? enUS;

  const sortedChildren = [...childRecords].sort((a, b) => {
    const nameA = `${a.first_name} ${a.last_name}`;
    const nameB = `${b.first_name} ${b.last_name}`;
    return nameA.localeCompare(nameB);
  });

  if (sortedChildren.length === 0) {
    return <p className="text-muted-foreground py-8 text-center">{t('noChildren')}</p>;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{tCommon('name')}</TableHead>
          {days.map((day) => (
            <TableHead key={day.toISOString()} className="text-center">
              {format(day, 'EEE dd.MM', { locale: dfLocale })}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {sortedChildren.map((child) => (
          <TableRow key={child.id}>
            <TableCell className="font-medium">
              {child.first_name} {child.last_name}
            </TableCell>
            {days.map((day) => {
              const dayStr = format(day, 'yyyy-MM-dd');
              const dayRecords = attendanceByDate.get(dayStr) ?? [];
              const attendance = dayRecords.find((a) => a.child_id === child.id);
              return (
                <TableCell key={dayStr} className="text-center">
                  <div className="flex justify-center">
                    <AttendanceCell
                      attendance={attendance}
                      childId={child.id}
                      dateStr={dayStr}
                      onCheckIn={onCheckIn}
                      onCheckOut={onCheckOut}
                      onUpdateTime={onUpdateTime}
                      onSetStatus={onSetStatus}
                      onSaveNote={onSaveNote}
                    />
                  </div>
                </TableCell>
              );
            })}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
