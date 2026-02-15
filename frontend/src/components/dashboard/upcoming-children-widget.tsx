'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
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
import type { ContractProperties } from '@/lib/api/types';
import { calculateAge, formatDate, propertiesToValues } from '@/lib/utils/formatting';
import { compareDates, toUTCDate } from '@/lib/utils/contracts';

interface UpcomingChildrenWidgetProps {
  orgId: number;
}

export function UpcomingChildrenWidget({ orgId }: UpcomingChildrenWidgetProps) {
  const t = useTranslations('upcomingChildren');
  const tChildren = useTranslations('children');
  const tContracts = useTranslations('contracts');
  const tGender = useTranslations('gender');

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
              <TableHead>{t('gender')}</TableHead>
              <TableHead>{t('birthdate')}</TableHead>
              <TableHead>{t('age')}</TableHead>
              <TableHead>{t('section')}</TableHead>
              <TableHead>{t('properties')}</TableHead>
              <TableHead>{t('startDate')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data
              .map((child) => {
                const futureContract = child.contracts
                  ?.filter((c) => toUTCDate(c.from) > Date.now())
                  .sort((a, b) => compareDates(a.from, b.from))[0];
                return { child, futureContract };
              })
              .sort((a, b) => {
                if (!a.futureContract) return 1;
                if (!b.futureContract) return -1;
                return compareDates(a.futureContract.from, b.futureContract.from);
              })
              .map(({ child, futureContract }) => (
                <TableRow key={child.id}>
                  <TableCell className="font-medium">
                    {child.first_name} {child.last_name}
                  </TableCell>
                  <TableCell>{tGender(child.gender)}</TableCell>
                  <TableCell>{formatDate(child.birthdate)}</TableCell>
                  <TableCell>{calculateAge(child.birthdate)}</TableCell>
                  <TableCell>
                    {futureContract?.section_name && <span>{futureContract.section_name}</span>}
                  </TableCell>
                  <TableCell>
                    {futureContract?.properties &&
                    Object.keys(futureContract.properties).length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {propertiesToValues(futureContract.properties as ContractProperties)
                          .slice(0, 3)
                          .map((value) => (
                            <Badge key={value} variant="outline" className="text-xs">
                              {value}
                            </Badge>
                          ))}
                        {Object.keys(futureContract.properties).length > 3 && (
                          <Badge variant="outline" className="text-xs">
                            +{Object.keys(futureContract.properties).length - 3}
                          </Badge>
                        )}
                      </div>
                    ) : (
                      <span className="text-sm text-muted-foreground">
                        {tContracts('noProperties')}
                      </span>
                    )}
                  </TableCell>
                  <TableCell>{futureContract ? formatDate(futureContract.from) : '-'}</TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
