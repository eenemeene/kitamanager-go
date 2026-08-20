'use client';

import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { classifySchoolEnrollment } from '@/lib/utils/school-enrollment';

export interface SchoolEnrollmentBadgeProps {
  birthdate: string;
  state?: string;
  /** A recorded school-entry date, when the birthdate is not the whole story. */
  schoolEntryDate?: string | null;
}

export function SchoolEnrollmentBadge({
  birthdate,
  state,
  schoolEntryDate,
}: SchoolEnrollmentBadgeProps) {
  const t = useTranslations();
  // No longer gated on a known state: a recorded date is an answer even where
  // there are no rules to derive one.
  const enrollment = classifySchoolEnrollment(birthdate, state ?? '', schoolEntryDate);
  if (!enrollment) return null;

  // Name the divergence rather than quietly showing a different year -- that a
  // decision was made is the part worth keeping when reconciling against ISBJ.
  const divergence =
    enrollment.overridden && enrollment.computedMussYear !== null
      ? enrollment.mussYear > enrollment.computedMussYear
        ? t('children.schoolEntryDeferred')
        : enrollment.mussYear < enrollment.computedMussYear
          ? t('children.schoolEntryEarly')
          : null
      : null;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex flex-wrap items-center gap-1">
          <Badge variant="secondary" className="text-xs">
            {enrollment.overridden
              ? t('children.schoolEntryYear', { year: enrollment.mussYear })
              : t('children.mussKind', { year: enrollment.mussYear })}
          </Badge>
          {divergence !== null && (
            <Badge variant="outline" className="text-xs">
              {divergence}
            </Badge>
          )}
          {enrollment.kannYear !== null && (
            <Badge variant="outline" className="text-xs">
              {t('children.kannKind', { year: enrollment.kannYear })}
            </Badge>
          )}
        </span>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">
        <p>
          {enrollment.overridden
            ? t('children.schoolEntryRecordedTooltip')
            : t('children.schoolEnrollmentTooltip')}
        </p>
      </TooltipContent>
    </Tooltip>
  );
}
