'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { EmployeeStaffingHoursResponse } from '@/lib/api/types';

interface EmployeeStaffingHoursTableProps {
  data: EmployeeStaffingHoursResponse;
}

function formatMonthHeader(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  return date.toLocaleDateString('de-DE', { month: 'short', year: '2-digit' });
}

function formatHours(value: number): string {
  if (value === 0) return '\u2013';
  return value.toLocaleString('de-DE', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

const STAFF_CATEGORY_LABELS: Record<string, string> = {
  qualified: 'Q',
  supplementary: 'S',
  non_pedagogical: 'NP',
};

export function EmployeeStaffingHoursTable({ data }: EmployeeStaffingHoursTableProps) {
  const t = useTranslations('statistics');

  const dates = data.dates ?? [];
  const employees = data.employees ?? [];

  const totals = useMemo(() => {
    const sums = new Array<number>(dates.length).fill(0);
    for (const emp of employees) {
      for (let i = 0; i < dates.length; i++) {
        sums[i] += emp.monthly_hours[i] ?? 0;
      }
    }
    return sums;
  }, [dates, employees]);

  const averages = useMemo(() => {
    return employees.map((emp) => {
      const hours = emp.monthly_hours ?? [];
      const nonZero = hours.filter((h) => h > 0);
      if (nonZero.length === 0) return 0;
      return nonZero.reduce((a, b) => a + b, 0) / nonZero.length;
    });
  }, [employees]);

  const totalAvg = useMemo(() => {
    const nonZero = totals.filter((h) => h > 0);
    if (nonZero.length === 0) return 0;
    return nonZero.reduce((a, b) => a + b, 0) / nonZero.length;
  }, [totals]);

  if (dates.length === 0 || employees.length === 0) {
    return <p className="text-muted-foreground">{t('chartError')}</p>;
  }

  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="bg-background sticky left-0 z-10 min-w-[180px]">
              {t('employeeName')}
            </TableHead>
            {dates.map((d) => (
              <TableHead key={d} className="min-w-[70px] text-right">
                {formatMonthHeader(d)}
              </TableHead>
            ))}
            <TableHead className="min-w-[70px] text-right font-bold">{t('average')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {employees.map((emp, empIdx) => (
            <TableRow key={emp.employee_id}>
              <TableCell className="bg-background sticky left-0 z-10">
                <div className="flex items-center gap-2">
                  <span className="font-medium">
                    {emp.last_name}, {emp.first_name}
                  </span>
                  {emp.staff_category && (
                    <span className="text-muted-foreground text-xs">
                      {STAFF_CATEGORY_LABELS[emp.staff_category] ?? emp.staff_category}
                    </span>
                  )}
                </div>
              </TableCell>
              {(emp.monthly_hours ?? []).map((hours, i) => (
                <TableCell key={dates[i]} className="text-right tabular-nums">
                  {formatHours(hours)}
                </TableCell>
              ))}
              <TableCell className="text-right font-bold tabular-nums">
                {formatHours(averages[empIdx])}
              </TableCell>
            </TableRow>
          ))}

          {/* Total row */}
          <TableRow className="border-t-2">
            <TableCell className="bg-background sticky left-0 z-10 font-bold">
              {t('total')}
            </TableCell>
            {totals.map((val, i) => (
              <TableCell key={dates[i]} className="text-right font-bold tabular-nums">
                {formatHours(val)}
              </TableCell>
            ))}
            <TableCell className="text-right font-bold tabular-nums">
              {formatHours(totalAvg)}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}
