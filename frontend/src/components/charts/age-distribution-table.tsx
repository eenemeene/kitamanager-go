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
import type { AgeDistributionResponse } from '@/lib/api/types';

interface AgeDistributionTableProps {
  data: AgeDistributionResponse;
}

/**
 * The age-distribution chart's numbers, as text.
 *
 * A stacked bar shows the shape of the distribution and nothing exact: the
 * count behind each segment lives in a tooltip, which needs a pointer to
 * open. This renders the same data the chart draws, so the page carries its
 * figures for anyone who cannot see the graphic — or who just wants to read
 * the numbers off.
 */
export function AgeDistributionTable({ data }: AgeDistributionTableProps) {
  const t = useTranslations();

  // "6+" is a bucket, not an age, so it gets its own label rather than being
  // fed through the "{age} years" message.
  const ageLabel = (raw: string) =>
    raw.includes('+') ? t('statistics.ageSixPlus') : t('statistics.ageYears', { age: raw });

  const totals = useMemo(
    () =>
      data.distribution.reduce(
        (acc, b) => ({
          male: acc.male + (b.male_count ?? 0),
          female: acc.female + (b.female_count ?? 0),
          diverse: acc.diverse + (b.diverse_count ?? 0),
          all: acc.all + (b.count ?? 0),
        }),
        { male: 0, female: 0, diverse: 0, all: 0 }
      ),
    [data.distribution]
  );

  return (
    <Table>
      <TableCaption className="sr-only">{t('statistics.ageDistributionTable')}</TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead>{t('statistics.ageGroup')}</TableHead>
          <TableHead className="text-right">{t('gender.male')}</TableHead>
          <TableHead className="text-right">{t('gender.female')}</TableHead>
          <TableHead className="text-right">{t('gender.diverse')}</TableHead>
          <TableHead className="text-right">{t('statistics.total')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {data.distribution.map((bucket) => (
          <TableRow key={bucket.age_label}>
            <TableCell className="font-medium">{ageLabel(bucket.age_label ?? '')}</TableCell>
            <TableCell className="text-right">{bucket.male_count ?? 0}</TableCell>
            <TableCell className="text-right">{bucket.female_count ?? 0}</TableCell>
            <TableCell className="text-right">{bucket.diverse_count ?? 0}</TableCell>
            <TableCell className="text-right font-medium">{bucket.count ?? 0}</TableCell>
          </TableRow>
        ))}
        <TableRow className="border-t-2">
          <TableCell className="font-medium">{t('statistics.total')}</TableCell>
          <TableCell className="text-right font-medium">{totals.male}</TableCell>
          <TableCell className="text-right font-medium">{totals.female}</TableCell>
          <TableCell className="text-right font-medium">{totals.diverse}</TableCell>
          <TableCell className="text-right font-medium">{totals.all}</TableCell>
        </TableRow>
      </TableBody>
    </Table>
  );
}
