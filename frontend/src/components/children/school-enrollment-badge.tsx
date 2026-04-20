'use client';

import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { classifySchoolEnrollment } from '@/lib/utils/school-enrollment';

export interface SchoolEnrollmentBadgeProps {
  birthdate: string;
  state?: string;
}

export function SchoolEnrollmentBadge({ birthdate, state }: SchoolEnrollmentBadgeProps) {
  const t = useTranslations();
  const enrollment = state ? classifySchoolEnrollment(birthdate, state) : null;
  if (!enrollment) return null;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex flex-wrap items-center gap-1">
          <Badge variant="secondary" className="text-xs">
            {t('children.mussKind', { year: enrollment.mussYear })}
          </Badge>
          {enrollment.kannYear !== null && (
            <Badge variant="outline" className="text-xs">
              {t('children.kannKind', { year: enrollment.kannYear })}
            </Badge>
          )}
        </span>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">
        <p>{t('children.schoolEnrollmentTooltip')}</p>
      </TooltipContent>
    </Tooltip>
  );
}
