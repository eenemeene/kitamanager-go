'use client';

import { useDraggable } from '@dnd-kit/core';
import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';
import type { Employee } from '@/lib/api/types';
import { getActiveContract } from '@/lib/utils/contracts';

export interface EmployeeCardProps {
  employee: Employee;
}

export function EmployeeCard({ employee }: EmployeeCardProps) {
  const t = useTranslations();
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `employee-${employee.id}`,
    data: { employee, type: 'employee' },
  });

  const fullName = `${employee.first_name} ${employee.last_name}`;
  const activeContract = getActiveContract(employee.contracts);
  const staffCategoryKey = activeContract?.staff_category ?? 'qualified';
  const weeklyHours = activeContract?.weekly_hours;

  return (
    <Card
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      className={cn(
        'cursor-grab border-blue-200 bg-blue-50 active:cursor-grabbing dark:border-blue-800 dark:bg-blue-950',
        isDragging && 'opacity-50'
      )}
    >
      <CardContent className="p-3">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-sm font-medium">{fullName}</span>
          <Badge variant="secondary" className="shrink-0 text-xs">
            {t(`employees.staffCategory.${staffCategoryKey}`)}
          </Badge>
        </div>
        {weeklyHours != null && (
          <p className="text-muted-foreground mt-1 text-xs">
            {weeklyHours}h / {t('employees.weeklyHours').toLowerCase()}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
