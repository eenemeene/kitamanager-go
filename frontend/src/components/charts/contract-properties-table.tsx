'use client';

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
import type { ContractPropertiesDistributionResponse } from '@/lib/api/types';

interface ContractPropertiesTableProps {
  data: ContractPropertiesDistributionResponse;
}

/**
 * The contract-properties chart's numbers, as text.
 *
 * The chart rotates its category labels 45 degrees and drops any that do not
 * fit, so the bars are readable as a ranking and not much else. The share
 * column is the one thing the chart cannot show at all: a bar's height is only
 * comparable to the other bars, not to the number of children overall.
 */
export function ContractPropertiesTable({ data }: ContractPropertiesTableProps) {
  const t = useTranslations();
  const fmt = useFormatters();

  const total = data.total_children ?? 0;

  return (
    <Table>
      <TableCaption className="sr-only">{t('statistics.contractPropertiesTable')}</TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead>{t('statistics.property')}</TableHead>
          <TableHead className="text-right">{t('statistics.childrenCount')}</TableHead>
          <TableHead className="text-right">{t('statistics.share')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {data.properties.map((property) => (
          <TableRow key={`${property.key}-${property.value}`}>
            <TableCell className="font-medium">
              {property.label || `${property.key}: ${property.value}`}
            </TableCell>
            <TableCell className="text-right">{property.count ?? 0}</TableCell>
            <TableCell className="text-right">
              {total > 0 ? fmt.percentage(((property.count ?? 0) / total) * 100, 1) : '—'}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
