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
import type { StaffingHoursResponse } from '@/lib/api/types';

interface StaffingHoursTableProps {
  data: StaffingHoursResponse;
}

function formatMonthHeader(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  return date.toLocaleDateString('de-DE', { month: 'short', year: '2-digit' });
}

function formatHours(value: number): string {
  if (value === 0) return '\u2013';
  return value.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function formatPercent(value: number): string {
  if (!isFinite(value)) return '\u2013';
  return (
    value.toLocaleString('de-DE', { minimumFractionDigits: 1, maximumFractionDigits: 1 }) + '%'
  );
}

function formatCount(value: number): string {
  if (value === 0) return '\u2013';
  return value.toLocaleString('de-DE', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

export function StaffingHoursTable({ data }: StaffingHoursTableProps) {
  const t = useTranslations('statistics');

  const dataPoints = data.data_points ?? [];

  const computed = useMemo(() => {
    const required = dataPoints.map((dp) => dp.required_hours);
    const available = dataPoints.map((dp) => dp.available_hours);
    const balance = dataPoints.map((dp) => dp.available_hours - dp.required_hours);
    const balancePercent = dataPoints.map((dp) =>
      dp.required_hours === 0
        ? NaN
        : ((dp.available_hours - dp.required_hours) / dp.required_hours) * 100
    );
    const children = dataPoints.map((dp) => dp.child_count);
    const staff = dataPoints.map((dp) => dp.staff_count);

    const avg = (arr: number[]) => {
      if (arr.length === 0) return 0;
      return arr.reduce((a, b) => a + b, 0) / arr.length;
    };

    const avgPercent = (balances: number[], requireds: number[]) => {
      if (requireds.length === 0) return NaN;
      const totalReq = requireds.reduce((a, b) => a + b, 0);
      if (totalReq === 0) return NaN;
      const totalBal = balances.reduce((a, b) => a + b, 0);
      return (totalBal / totalReq) * 100;
    };

    return {
      required,
      available,
      balance,
      balancePercent,
      children,
      staff,
      avgRequired: avg(required),
      avgAvailable: avg(available),
      avgBalance: avg(balance),
      avgBalancePercent: avgPercent(balance, required),
      avgChildren: avg(children),
      avgStaff: avg(staff),
    };
  }, [dataPoints]);

  if (dataPoints.length === 0) {
    return <p className="text-muted-foreground">{t('chartError')}</p>;
  }

  const months = dataPoints.map((dp) => dp.date);

  const balanceColor = (val: number) =>
    val >= 0 ? 'text-green-700 dark:text-green-400' : 'text-red-700 dark:text-red-400';

  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="bg-background sticky left-0 z-10 min-w-[120px]" />
            {months.map((m) => (
              <TableHead key={m} className="min-w-[80px] text-right">
                {formatMonthHeader(m)}
              </TableHead>
            ))}
            <TableHead className="min-w-[80px] text-right font-bold">{t('average')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {/* Required */}
          <TableRow>
            <TableCell className="bg-background sticky left-0 z-10 font-medium">
              {t('staffingRequired')}
            </TableCell>
            {computed.required.map((val, i) => (
              <TableCell key={months[i]} className="text-right tabular-nums">
                {formatHours(val)}
              </TableCell>
            ))}
            <TableCell className="text-right font-bold tabular-nums">
              {formatHours(computed.avgRequired)}
            </TableCell>
          </TableRow>

          {/* Available */}
          <TableRow>
            <TableCell className="bg-background sticky left-0 z-10 font-medium">
              {t('staffingAvailable')}
            </TableCell>
            {computed.available.map((val, i) => (
              <TableCell key={months[i]} className="text-right tabular-nums">
                {formatHours(val)}
              </TableCell>
            ))}
            <TableCell className="text-right font-bold tabular-nums">
              {formatHours(computed.avgAvailable)}
            </TableCell>
          </TableRow>

          {/* Balance */}
          <TableRow className="border-t-2">
            <TableCell className="bg-background sticky left-0 z-10 font-medium">
              {t('staffingBalance')}
            </TableCell>
            {computed.balance.map((val, i) => (
              <TableCell
                key={months[i]}
                className={`text-right font-bold tabular-nums ${balanceColor(val)}`}
              >
                {formatHours(val)}
              </TableCell>
            ))}
            <TableCell
              className={`text-right font-bold tabular-nums ${balanceColor(computed.avgBalance)}`}
            >
              {formatHours(computed.avgBalance)}
            </TableCell>
          </TableRow>

          {/* Balance % */}
          <TableRow>
            <TableCell className="bg-background sticky left-0 z-10 font-medium">
              {t('staffingBalancePercent')}
            </TableCell>
            {computed.balancePercent.map((val, i) => (
              <TableCell
                key={months[i]}
                className={`text-right tabular-nums ${isFinite(val) ? balanceColor(val) : ''}`}
              >
                {formatPercent(val)}
              </TableCell>
            ))}
            <TableCell
              className={`text-right font-bold tabular-nums ${isFinite(computed.avgBalancePercent) ? balanceColor(computed.avgBalancePercent) : ''}`}
            >
              {formatPercent(computed.avgBalancePercent)}
            </TableCell>
          </TableRow>

          {/* Children */}
          <TableRow className="border-t-2">
            <TableCell className="bg-background sticky left-0 z-10 font-medium">
              {t('childrenContractCount')}
            </TableCell>
            {computed.children.map((val, i) => (
              <TableCell key={months[i]} className="text-right tabular-nums">
                {val || '\u2013'}
              </TableCell>
            ))}
            <TableCell className="text-right font-bold tabular-nums">
              {formatCount(computed.avgChildren)}
            </TableCell>
          </TableRow>

          {/* Staff */}
          <TableRow>
            <TableCell className="bg-background sticky left-0 z-10 font-medium">
              {t('staffCount')}
            </TableCell>
            {computed.staff.map((val, i) => (
              <TableCell key={months[i]} className="text-right tabular-nums">
                {val || '\u2013'}
              </TableCell>
            ))}
            <TableCell className="text-right font-bold tabular-nums">
              {formatCount(computed.avgStaff)}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}
