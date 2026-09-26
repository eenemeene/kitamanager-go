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
  /**
   * Whether this card can be picked up.
   *
   * False on a past or future snapshot of the board, where the only write it
   * could make -- amend the contract from today -- does not describe what is on
   * screen. Also false for the copy rendered inside the DragOverlay, which is a
   * picture of the card being dragged and must never take input of its own.
   */
  draggable?: boolean;
  /** The board's snapshot date, "YYYY-MM-DD" — which contract describes this card. */
  asOf?: string;
}

export function EmployeeCard({ employee, draggable = true, asOf }: EmployeeCardProps) {
  const t = useTranslations();
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `employee-${employee.id}`,
    data: { employee, type: 'employee' },
    disabled: !draggable,
  });

  const fullName = `${employee.first_name} ${employee.last_name}`;
  const activeContract = getActiveContract(employee.contracts, asOf);
  const staffCategoryKey = activeContract?.staff_category ?? 'qualified';
  const weeklyHours = activeContract?.weekly_hours;

  return (
    <Card
      ref={setNodeRef}
      // See ChildCard: a disabled draggable keeps dnd-kit's role and tab stop
      // unless the props are withheld.
      {...(draggable ? listeners : {})}
      {...(draggable ? attributes : {})}
      className={cn(
        'border-info/30 bg-info/10',
        draggable && 'cursor-grab active:cursor-grabbing',
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
