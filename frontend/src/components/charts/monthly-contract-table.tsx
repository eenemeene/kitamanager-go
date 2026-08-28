'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useFormatters } from '@/hooks/use-formatters';
import type { OccupancyResponse, StaffingHoursResponse } from '@/lib/api/types';

interface MonthlyContractTableProps {
  data: StaffingHoursResponse;
  occupancy?: OccupancyResponse;
}

/**
 * The monthly-contract chart's numbers, as text.
 *
 * The line gives the trend; the per-age-group split behind each month is only
 * in the hover tooltip. Both are here, because the split is the part a manager
 * actually acts on — a flat total can hide a group emptying while another
 * fills. The age-group columns appear only when the occupancy data they come
 * from has loaded, so the table degrades to the total rather than to nothing.
 */
export function MonthlyContractTable({ data, occupancy }: MonthlyContractTableProps) {
  const t = useTranslations();
  const fmt = useFormatters();

  const ageGroups = useMemo(
    () => occupancy?.age_groups.map((group) => group.label ?? '') ?? [],
    [occupancy]
  );

  // The occupancy response nests care types under each age group; the chart
  // sums them for its tooltip and so does this.
  const byDate = useMemo(() => {
    const map = new Map<string, Record<string, number>>();
    for (const point of occupancy?.data_points ?? []) {
      const perGroup: Record<string, number> = {};
      for (const group of ageGroups) {
        perGroup[group] = Object.values(point.by_age_and_care_type?.[group] ?? {}).reduce<number>(
          (sum, n) => sum + (n ?? 0),
          0
        );
      }
      map.set(point.date ?? '', perGroup);
    }
    return map;
  }, [occupancy, ageGroups]);

  return (
    <Table>
      <TableCaption className="sr-only">{t('statistics.monthlyContractsTable')}</TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead>{t('statistics.month')}</TableHead>
          <TableHead className="text-right">{t('statistics.childrenCount')}</TableHead>
          {ageGroups.map((group) => (
            <TableHead key={group} className="hidden text-right md:table-cell">
              {group}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {data.data_points.map((point) => {
          const date = point.date ?? '';
          const perGroup = byDate.get(date);
          return (
            <TableRow key={date}>
              <TableCell className="font-medium whitespace-nowrap">{fmt.monthYear(date)}</TableCell>
              <TableCell className="text-right">{point.child_count ?? 0}</TableCell>
              {ageGroups.map((group) => (
                <TableCell key={group} className="hidden text-right md:table-cell">
                  {perGroup?.[group] ?? 0}
                </TableCell>
              ))}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
