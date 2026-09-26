'use client';

import { useDraggable } from '@dnd-kit/core';
import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';
import { calculateAge } from '@/lib/utils/formatting';
import type { Child } from '@/lib/api/types';

export interface ChildCardProps {
  child: Child;
  /**
   * Whether this card can be picked up.
   *
   * False on a past or future snapshot of the board, where the only write it
   * could make -- amend the contract from today -- does not describe what is on
   * screen. Also false for the copy rendered inside the DragOverlay, which is a
   * picture of the card being dragged and must never take input of its own.
   */
  draggable?: boolean;
}

export function ChildCard({ child, draggable = true }: ChildCardProps) {
  const t = useTranslations();
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `child-${child.id}`,
    data: { child, type: 'child' },
    disabled: !draggable,
  });

  const age = calculateAge(child.birthdate);
  const fullName = `${child.first_name} ${child.last_name}`;
  const genderKey =
    child.gender === 'male' ? 'male' : child.gender === 'female' ? 'female' : 'diverse';

  return (
    <Card
      ref={setNodeRef}
      // Not spread when disabled: dnd-kit still sets role, tabIndex and
      // aria-roledescription="draggable" on a disabled draggable, so a card on a
      // read-only snapshot would keep announcing itself as movable and keep its
      // tab stop while space and enter did nothing.
      {...(draggable ? listeners : {})}
      {...(draggable ? attributes : {})}
      className={cn(draggable && 'cursor-grab active:cursor-grabbing', isDragging && 'opacity-50')}
    >
      <CardContent className="p-3">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-sm font-medium">{fullName}</span>
          <Badge variant="outline" className="shrink-0 text-xs" title={t(`gender.${genderKey}`)}>
            {/* The badge is one letter for width; a screen reader gets the word. */}
            <span aria-hidden="true">{t(`gender.short.${genderKey}`)}</span>
            <span className="sr-only">{t(`gender.${genderKey}`)}</span>
          </Badge>
        </div>
        <p className="text-muted-foreground mt-1 text-xs">{t('sections.childAge', { age })}</p>
      </CardContent>
    </Card>
  );
}
