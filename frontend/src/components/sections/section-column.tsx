'use client';

import { useDroppable } from '@dnd-kit/core';
import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { formatMonthRange } from '@/lib/utils/formatting';
import type { Child, Employee } from '@/lib/api/types';
import { ChildCard } from './child-card';
import { EmployeeCard } from './employee-card';

export interface SectionColumnProps {
  id: string;
  title: string;
  items: Child[];
  employees: Employee[];
  isDefault?: boolean;
  minAgeMonths?: number | null;
  maxAgeMonths?: number | null;
}

export function SectionColumn({
  id,
  title,
  items,
  employees,
  isDefault,
  minAgeMonths,
  maxAgeMonths,
}: SectionColumnProps) {
  const t = useTranslations();
  const { setNodeRef, isOver } = useDroppable({ id });

  const totalCount = employees.length + items.length;
  const ageRange = formatMonthRange(minAgeMonths, maxAgeMonths);

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'bg-muted/50 flex w-72 shrink-0 flex-col rounded-lg border transition-colors',
        isOver && 'border-primary bg-primary/5'
      )}
    >
      <div className="flex items-center justify-between border-b p-3">
        <div className="flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold">{title}</h3>
            {isDefault && (
              <Badge variant="secondary" className="text-xs">
                {t('sections.defaultSection')}
              </Badge>
            )}
          </div>
          {ageRange && (
            <span className="text-muted-foreground text-xs">
              {ageRange} {t('sections.months')}
            </span>
          )}
        </div>
        <Badge variant="outline" className="text-xs">
          {totalCount}
        </Badge>
      </div>
      <div className="flex flex-1 flex-col gap-2 overflow-y-auto p-2">
        {totalCount === 0 ? (
          <p className="text-muted-foreground py-4 text-center text-xs">{t('common.noResults')}</p>
        ) : (
          <>
            {employees.map((employee) => (
              <EmployeeCard key={`emp-${employee.id}`} employee={employee} />
            ))}
            {employees.length > 0 && items.length > 0 && (
              <div className="my-1 border-t border-dashed" />
            )}
            {items.map((child) => (
              <ChildCard key={child.id} child={child} />
            ))}
          </>
        )}
      </div>
    </div>
  );
}
