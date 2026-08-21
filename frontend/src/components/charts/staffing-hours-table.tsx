'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { HeaderWithTooltip } from '@/components/ui/header-with-tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { TooltipProvider } from '@/components/ui/tooltip';
import type { StaffingHoursResponse } from '@/lib/api/types';
import { useFormatters } from '@/hooks/use-formatters';

interface StaffingHoursTableProps {
  data: StaffingHoursResponse;
}

export function StaffingHoursTable({ data }: StaffingHoursTableProps) {
  const t = useTranslations('statistics');
  const fmt = useFormatters();
  const formatMonthHeader = (dateStr: string) => fmt.monthYear(dateStr);
  const formatHours = (value: number) =>
    value === 0
      ? '\u2013'
      : fmt.number(value, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  const formatPercent = (value: number) => (isFinite(value) ? fmt.percentage(value, 1) : '\u2013');
  const formatCount = (value: number) =>
    value === 0
      ? '\u2013'
      : fmt.number(value, { minimumFractionDigits: 1, maximumFractionDigits: 1 });

  // Memoised so the `?? []` fallback does not produce a fresh array on every
  // render, which would invalidate the useMemo below that depends on it.
  const dataPoints = useMemo(() => data.data_points ?? [], [data.data_points]);

  const computed = useMemo(() => {
    const required = dataPoints.map((dp) => dp.required_hours ?? 0);
    const available = dataPoints.map((dp) => dp.available_hours ?? 0);
    const balance = dataPoints.map((dp) => (dp.available_hours ?? 0) - (dp.required_hours ?? 0));
    const balancePercent = dataPoints.map((dp) => {
      const req = dp.required_hours ?? 0;
      const avail = dp.available_hours ?? 0;
      return req === 0 ? NaN : ((avail - req) / req) * 100;
    });
    const children = dataPoints.map((dp) => dp.child_count ?? 0);
    const staff = dataPoints.map((dp) => dp.staff_count ?? 0);

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

  const months = dataPoints.map((dp) => dp.date ?? '');

  const balanceColor = (val: number) => (val >= 0 ? 'text-success' : 'text-destructive');

  return (
    <TooltipProvider>
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
                <HeaderWithTooltip
                  label={t('staffingRequired')}
                  tooltip={t('requiredHoursTooltip')}
                />
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
                <HeaderWithTooltip
                  label={t('staffingAvailable')}
                  tooltip={t('availableHoursTooltip')}
                />
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
                <HeaderWithTooltip
                  label={t('staffingBalance')}
                  tooltip={t('staffingBalanceTooltip')}
                />
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
                <HeaderWithTooltip
                  label={t('staffingBalancePercent')}
                  tooltip={t('balancePercentageTooltip')}
                />
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
    </TooltipProvider>
  );
}
