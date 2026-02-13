'use client';

import { useMemo } from 'react';
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
import { getActiveContract } from '@/lib/utils/contracts';

const ALERT_THRESHOLD_MONTHS = 3;

interface AgeAlert {
  childId: number;
  childName: string;
  sectionName: string;
  ageMonths: number;
  maxAgeMonths: number;
  monthsRemaining: number;
}

interface SectionAgeAlertsWidgetProps {
  orgId: number;
}

export function SectionAgeAlertsWidget({ orgId }: SectionAgeAlertsWidgetProps) {
  const t = useTranslations('sectionAgeAlerts');

  const { data: children } = useQuery({
    queryKey: queryKeys.children.allUnpaginated(orgId),
    queryFn: () => apiClient.getChildrenAll(orgId),
    enabled: !!orgId,
  });

  const { data: sectionsData } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: 100 }),
    enabled: !!orgId,
  });

  const alerts = useMemo<AgeAlert[]>(() => {
    if (!children || !sectionsData?.data) return [];

    const sections = sectionsData.data;
    const sectionMap = new Map(sections.map((s) => [s.id, s]));
    const now = Date.now();
    const result: AgeAlert[] = [];

    for (const child of children) {
      if (!child.birthdate) continue;
      const activeContract = getActiveContract(child.contracts);
      if (!activeContract?.section_id) continue;
      const section = sectionMap.get(activeContract.section_id);
      if (!section || section.max_age_months == null) continue;

      const ageMonths = Math.floor(
        (now - new Date(child.birthdate).getTime()) / (1000 * 60 * 60 * 24 * 30.44)
      );
      const monthsRemaining = section.max_age_months - ageMonths;

      if (monthsRemaining <= ALERT_THRESHOLD_MONTHS) {
        result.push({
          childId: child.id,
          childName: `${child.first_name} ${child.last_name}`,
          sectionName: section.name,
          ageMonths,
          maxAgeMonths: section.max_age_months,
          monthsRemaining,
        });
      }
    }

    result.sort((a, b) => a.monthsRemaining - b.monthsRemaining);
    return result;
  }, [children, sectionsData]);

  if (alerts.length === 0) {
    return null;
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-base font-medium">{t('title')}</CardTitle>
        <Badge variant="secondary">{t('count', { count: alerts.length })}</Badge>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('child')}</TableHead>
              <TableHead>{t('section')}</TableHead>
              <TableHead className="text-right">{t('ageMonths')}</TableHead>
              <TableHead className="text-right">{t('maxAge')}</TableHead>
              <TableHead className="text-right">{t('remaining')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {alerts.map((alert) => (
              <TableRow key={alert.childId}>
                <TableCell className="font-medium">{alert.childName}</TableCell>
                <TableCell>{alert.sectionName}</TableCell>
                <TableCell className="text-right">{alert.ageMonths}</TableCell>
                <TableCell className="text-right">{alert.maxAgeMonths}</TableCell>
                <TableCell className="text-right">
                  {alert.monthsRemaining <= 0 ? (
                    <Badge variant="destructive">{t('overdue')}</Badge>
                  ) : (
                    <Badge variant="secondary">
                      {t('monthsLeft', { count: alert.monthsRemaining })}
                    </Badge>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
