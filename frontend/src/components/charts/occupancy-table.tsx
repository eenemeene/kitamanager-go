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
import type { OccupancyResponse } from '@/lib/api/types';

interface OccupancyTableProps {
  data: OccupancyResponse;
}

function formatMonthHeader(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  return date.toLocaleDateString('de-DE', { month: 'short', year: '2-digit' });
}

export function OccupancyTable({ data }: OccupancyTableProps) {
  const t = useTranslations('statistics');

  const months = useMemo(() => data.data_points.map((dp) => dp.date), [data.data_points]);

  // Build rows: one per (age group × care type) combination
  const matrixRows = useMemo(() => {
    const rows: {
      ageLabel: string;
      careTypeLabel: string;
      ageGroupIndex: number;
      values: number[];
    }[] = [];
    for (let agIdx = 0; agIdx < data.age_groups.length; agIdx++) {
      const ag = data.age_groups[agIdx];
      for (const ct of data.care_types) {
        const values = data.data_points.map(
          (dp) => dp.by_age_and_care_type?.[ag.label]?.[ct.value] ?? 0
        );
        rows.push({
          ageLabel: ag.label,
          careTypeLabel: ct.label || ct.value,
          ageGroupIndex: agIdx,
          values,
        });
      }
    }
    return rows;
  }, [data]);

  // Total row values
  const totalValues = useMemo(() => data.data_points.map((dp) => dp.total), [data.data_points]);

  // Supplement rows
  const supplementRows = useMemo(() => {
    return data.supplement_types.map((st) => ({
      label: st.label,
      values: data.data_points.map((dp) => dp.by_supplement?.[st.value] ?? 0),
    }));
  }, [data]);

  return (
    <div className="overflow-x-auto text-xs">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="bg-background sticky left-0 z-10 min-w-[60px] px-2 py-1">
              {t('ageGroup')}
            </TableHead>
            <TableHead className="bg-background sticky left-[60px] z-10 min-w-[80px] px-2 py-1">
              {t('careType')}
            </TableHead>
            {months.map((m) => (
              <TableHead key={m} className="min-w-[50px] px-1 py-1 text-center">
                {formatMonthHeader(m)}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {/* Age group × care type rows */}
          {matrixRows.map((row, idx) => {
            // Show age group label only on first row of each group
            const isFirstInGroup = idx === 0 || matrixRows[idx - 1].ageLabel !== row.ageLabel;
            const rowsInGroup = matrixRows.filter((r) => r.ageLabel === row.ageLabel).length;
            const isEvenGroup = row.ageGroupIndex % 2 === 0;
            const rowBg = isEvenGroup ? 'bg-muted/50' : 'bg-background';

            return (
              <TableRow
                key={`${row.ageLabel}-${row.careTypeLabel}`}
                className={`${rowBg}${isFirstInGroup && idx > 0 ? 'border-t-2' : ''}`}
              >
                {isFirstInGroup ? (
                  <TableCell
                    className={`sticky left-0 z-10 px-2 py-1 ${rowBg} font-medium`}
                    rowSpan={rowsInGroup}
                  >
                    {row.ageLabel}
                  </TableCell>
                ) : null}
                <TableCell className={`sticky left-[60px] z-10 px-2 py-1 ${rowBg}`}>
                  {row.careTypeLabel}
                </TableCell>
                {row.values.map((val, i) => (
                  <TableCell key={months[i]} className="px-1 py-1 text-center tabular-nums">
                    {val || '\u2013'}
                  </TableCell>
                ))}
              </TableRow>
            );
          })}

          {/* Total row */}
          <TableRow className="border-t-2 font-bold">
            <TableCell className="bg-background sticky left-0 z-10 px-2 py-1" colSpan={2}>
              {t('total')}
            </TableCell>
            {totalValues.map((val, i) => (
              <TableCell key={months[i]} className="px-1 py-1 text-center tabular-nums">
                {val || '\u2013'}
              </TableCell>
            ))}
          </TableRow>

          {/* Supplements section */}
          {supplementRows.length > 0 && (
            <>
              <TableRow>
                <TableCell
                  className="bg-background text-muted-foreground sticky left-0 z-10 px-2 py-2 font-medium"
                  colSpan={2 + months.length}
                >
                  {t('supplements')}
                </TableCell>
              </TableRow>
              {supplementRows.map((row) => (
                <TableRow key={row.label}>
                  <TableCell className="bg-background sticky left-0 z-10 px-2 py-1" colSpan={2}>
                    {row.label}
                  </TableCell>
                  {row.values.map((val, i) => (
                    <TableCell key={months[i]} className="px-1 py-1 text-center tabular-nums">
                      {val || '\u2013'}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
