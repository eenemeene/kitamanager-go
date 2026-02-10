'use client';

import { useDroppable } from '@dnd-kit/core';
import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import type { Child } from '@/lib/api/types';
import { ChildCard } from './child-card';

export interface SectionColumnProps {
  id: string;
  title: string;
  items: Child[];
  isDefault?: boolean;
}

export function SectionColumn({ id, title, items, isDefault }: SectionColumnProps) {
  const t = useTranslations();
  const { setNodeRef, isOver } = useDroppable({ id });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'flex w-72 shrink-0 flex-col rounded-lg border bg-muted/50 transition-colors',
        isOver && 'border-primary bg-primary/5'
      )}
    >
      <div className="flex items-center justify-between border-b p-3">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold">{title}</h3>
          {isDefault && (
            <Badge variant="secondary" className="text-xs">
              {t('sections.defaultSection')}
            </Badge>
          )}
        </div>
        <Badge variant="outline" className="text-xs">
          {items.length}
        </Badge>
      </div>
      <div className="flex flex-1 flex-col gap-2 overflow-y-auto p-2">
        {items.length === 0 ? (
          <p className="py-4 text-center text-xs text-muted-foreground">{t('common.noResults')}</p>
        ) : (
          items.map((child) => <ChildCard key={child.id} child={child} />)
        )}
      </div>
    </div>
  );
}
