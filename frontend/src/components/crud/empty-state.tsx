'use client';

import { ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import { LucideIcon } from 'lucide-react';

export interface EmptyStateProps {
  /** Lucide icon component to display above the title */
  icon: LucideIcon;
  /** i18n key for the title, or a raw string (raw if it contains a space) */
  title: string;
  /** i18n key for the description, or a raw string (raw if it contains a space) */
  description: string;
  /** Optional action buttons rendered below the description */
  action?: ReactNode;
}

/**
 * Friendly empty-state panel for first-time users of a resource page.
 * Use when there is no data AND the user has applied no filters/search;
 * for filtered-but-empty results stick with the table's plain "no results"
 * row, since changing filters is the resolution there.
 */
export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  const t = useTranslations();

  const titleText = title.includes(' ') ? title : t(title);
  const descriptionText = description.includes(' ') ? description : t(description);

  return (
    <div className="flex flex-col items-center justify-center gap-3 px-4 py-12 text-center">
      <Icon className="text-muted-foreground h-10 w-10" aria-hidden="true" />
      <div className="space-y-1">
        <h3 className="text-base font-semibold">{titleText}</h3>
        <p className="text-muted-foreground max-w-md text-sm">{descriptionText}</p>
      </div>
      {action && <div className="flex flex-wrap justify-center gap-2">{action}</div>}
    </div>
  );
}
