'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { formatDate } from '@/lib/utils/formatting';
import { compareDates, toUTCDate } from '@/lib/utils/contracts';

interface UpcomingChildrenWidgetProps {
  orgId: number;
}

export function UpcomingChildrenWidget({ orgId }: UpcomingChildrenWidgetProps) {
  const t = useTranslations('upcomingChildren');

  const { data } = useQuery({
    queryKey: queryKeys.children.upcoming(orgId),
    queryFn: () => apiClient.getUpcomingChildren(orgId),
    enabled: !!orgId,
  });

  if (!data || data.length === 0) {
    return null;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base font-medium">{t('title')}</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('name')}</TableHead>
              <TableHead>{t('section')}</TableHead>
              <TableHead>{t('startDate')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((child) => {
              // Pick the earliest future contract (contracts are preloaded, find the one starting soonest)
              const futureContract = child.contracts
                ?.filter((c) => toUTCDate(c.from) > Date.now())
                .sort((a, b) => compareDates(a.from, b.from))[0];
              return (
                <TableRow key={child.id}>
                  <TableCell className="font-medium">
                    {child.first_name} {child.last_name}
                  </TableCell>
                  <TableCell>{futureContract?.section_name ?? '-'}</TableCell>
                  <TableCell>{futureContract ? formatDate(futureContract.from) : '-'}</TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
