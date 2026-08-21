'use client';

import { useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { ArrowRight } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
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
import { getActiveContract, todayBerlinString } from '@/lib/utils/contracts';
import { type Section, LOOKUP_FETCH_LIMIT } from '@/lib/api/types';

interface AgeAlert {
  childId: number;
  childName: string;
  contractId: number;
  /** The contract version as read, sent as the If-Match precondition. */
  contractVersion: number;
  /** The contract's start date, which decides amend versus correct. */
  contractFrom: string;
  sectionName: string;
  ageMonths: number;
  maxAgeMonths: number;
  nextSection: Section | null;
}

interface SectionAgeAlertsWidgetProps {
  orgId: number;
}

function findNextSection(
  ageMonths: number,
  currentSectionId: number,
  sections: Section[]
): Section | null {
  // Find sections where the child's age fits within the range,
  // excluding the current section.
  const candidates = sections.filter((s) => {
    if (s.id === currentSectionId) return false;
    if (s.min_age_months == null) return false;
    const minOk = ageMonths >= s.min_age_months;
    const maxOk = s.max_age_months == null || ageMonths < s.max_age_months;
    return minOk && maxOk;
  });

  if (candidates.length === 0) return null;

  // Pick the one with the closest (lowest) min_age_months above or at the child's age.
  // This gives the "next" section rather than a much older section.
  candidates.sort((a, b) => (a.min_age_months ?? 0) - (b.min_age_months ?? 0));
  return candidates[0];
}

export function SectionAgeAlertsWidget({ orgId }: SectionAgeAlertsWidgetProps) {
  const t = useTranslations('sectionAgeAlerts');
  const queryClient = useQueryClient();

  const { data: children } = useQuery({
    queryKey: queryKeys.children.allUnpaginated(orgId),
    queryFn: () => apiClient.getChildrenAll(orgId),
    enabled: !!orgId,
  });

  const { data: sectionsData } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const moveMutation = useMutation({
    mutationFn: async ({
      childId,
      contractId,
      sectionId,
      version,
      from,
    }: {
      childId: number;
      contractId: number;
      sectionId: number;
      version: number;
      from: string;
    }): Promise<void> => {
      // Amend rather than correct: which section a child was in is history that
      // occupancy and staffing reports read. A contract that has not started yet
      // has no such history, and an amendment of it would be refused anyway,
      // because an effective date has to fall after the contract's own start.
      const today = todayBerlinString();
      if (from.slice(0, 10) > today) {
        await apiClient.correctChildContract(orgId, childId, contractId, version, {
          section_id: sectionId,
        });
        return;
      }
      await apiClient.amendChildContract(orgId, childId, contractId, version, {
        effective_from: today,
        section_id: sectionId,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.allUnpaginated(orgId) });
    },
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

      if (monthsRemaining <= 0) {
        result.push({
          childId: child.id,
          childName: `${child.first_name} ${child.last_name}`,
          contractId: activeContract.id,
          contractVersion: activeContract.version,
          contractFrom: activeContract.from,
          sectionName: section.name,
          ageMonths,
          maxAgeMonths: section.max_age_months,
          nextSection: findNextSection(ageMonths, section.id, sections),
        });
      }
    }

    result.sort((a, b) => b.ageMonths - b.maxAgeMonths - (a.ageMonths - a.maxAgeMonths));
    return result;
  }, [children, sectionsData]);

  if (alerts.length === 0) {
    return null;
  }

  return (
    <Card>
      <CardHeader className="space-y-1 pb-2">
        <div className="flex flex-row items-center justify-between">
          <CardTitle className="text-base font-medium">{t('title')}</CardTitle>
          <Badge variant="secondary">{t('count', { count: alerts.length })}</Badge>
        </div>
        <CardDescription>{t('description')}</CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('child')}</TableHead>
              <TableHead>{t('section')}</TableHead>
              <TableHead className="text-right">{t('ageMonths')}</TableHead>
              <TableHead className="text-right">{t('maxAge')}</TableHead>
              <TableHead></TableHead>
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
                  {alert.nextSection ? (
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={moveMutation.isPending}
                      onClick={() =>
                        moveMutation.mutate({
                          childId: alert.childId,
                          contractId: alert.contractId,
                          sectionId: alert.nextSection!.id,
                          version: alert.contractVersion,
                          from: alert.contractFrom,
                        })
                      }
                    >
                      <ArrowRight className="mr-1 h-3 w-3" />
                      {t('moveTo', { section: alert.nextSection.name })}
                    </Button>
                  ) : (
                    <Badge variant="destructive">{t('overdue')}</Badge>
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
