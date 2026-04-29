'use client';

import { useMemo, Fragment } from 'react';
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

function formatDiff(value: number): string {
  return value.toLocaleString('de-DE', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

function DiffCell({ prev, curr }: { prev: number; curr: number }) {
  // Both zero → no meaningful change
  if (prev === 0 && curr === 0) return <TableCell className="px-0 text-center text-[10px]" />;
  const diff = curr - prev;
  if (diff === 0) return <TableCell className="px-0 text-center text-[10px]" />;
  const isUp = diff > 0;
  return (
    <TableCell className="px-0 text-center text-[10px] whitespace-nowrap">
      <span className={isUp ? 'text-success' : 'text-destructive'}>
        {isUp ? '▲' : '▼'} {isUp ? '+' : ''}
        {formatDiff(diff)}
      </span>
    </TableCell>
  );
}

const STAFF_CATEGORY_LABELS: Record<string, string> = {
  qualified: 'Q',
  supplementary: 'S',
  non_pedagogical: 'NP',
};

const STAFF_CATEGORY_TRANSLATION_KEYS: Record<string, string> = {
  qualified: 'employees.staffCategory.qualified',
  supplementary: 'employees.staffCategory.supplementary',
  non_pedagogical: 'employees.staffCategory.non_pedagogical',
};

export function EmployeeStaffingHoursTable({ data }: EmployeeStaffingHoursTableProps) {
  const t = useTranslations('statistics');
  const tRoot = useTranslations();

  const dates = data.dates ?? [];
  const employees = data.employees ?? [];

  const totals = useMemo(() => {
    const sums = new Array<number>(dates.length).fill(0);
    for (const emp of employees) {
      for (let i = 0; i < dates.length; i++) {
        sums[i] += emp.monthly_hours?.[i] ?? 0;
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

  const hasCategories = employees.some((e) => e.staff_category);

  return (
    <div className="overflow-x-auto">
      {hasCategories && (
        <p className="text-muted-foreground mb-2 text-xs">
          <span className="font-medium">Q</span> = {tRoot('employees.staffCategory.qualified')} ·{' '}
          <span className="font-medium">S</span> = {tRoot('employees.staffCategory.supplementary')}{' '}
          · <span className="font-medium">NP</span> ={' '}
          {tRoot('employees.staffCategory.non_pedagogical')}
        </p>
      )}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="bg-background sticky left-0 z-10 min-w-[180px]">
              {t('employeeName')}
            </TableHead>
            {dates.map((d, i) => (
              <Fragment key={d}>
                {i > 0 && <TableHead className="w-0 px-0" />}
                <TableHead className="min-w-[70px] text-right">{formatMonthHeader(d)}</TableHead>
              </Fragment>
            ))}
            <TableHead className="min-w-[70px] text-right font-bold">{t('average')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {employees.map((emp, empIdx) => {
            const hours = emp.monthly_hours ?? [];
            return (
              <TableRow key={emp.employee_id}>
                <TableCell className="bg-background sticky left-0 z-10">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">
                      {emp.last_name}, {emp.first_name}
                    </span>
                    {emp.staff_category && (
                      <span
                        className="text-muted-foreground cursor-help border-b border-dotted border-current text-xs"
                        title={
                          STAFF_CATEGORY_TRANSLATION_KEYS[emp.staff_category]
                            ? tRoot(
                                STAFF_CATEGORY_TRANSLATION_KEYS[emp.staff_category] as Parameters<
                                  typeof tRoot
                                >[0]
                              )
                            : emp.staff_category
                        }
                      >
                        {STAFF_CATEGORY_LABELS[emp.staff_category] ?? emp.staff_category}
                      </span>
                    )}
                  </div>
                </TableCell>
                {hours.map((h, i) => (
                  <Fragment key={dates[i]}>
                    {i > 0 && <DiffCell prev={hours[i - 1]} curr={h} />}
                    <TableCell className="text-right tabular-nums">{formatHours(h)}</TableCell>
                  </Fragment>
                ))}
                <TableCell className="text-right font-bold tabular-nums">
                  {formatHours(averages[empIdx])}
                </TableCell>
              </TableRow>
            );
          })}

          {/* Total row */}
          <TableRow className="border-t-2">
            <TableCell className="bg-background sticky left-0 z-10 font-bold">
              {t('total')}
            </TableCell>
            {totals.map((val, i) => (
              <Fragment key={dates[i]}>
                {i > 0 && <DiffCell prev={totals[i - 1]} curr={val} />}
                <TableCell className="text-right font-bold tabular-nums">
                  {formatHours(val)}
                </TableCell>
              </Fragment>
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
