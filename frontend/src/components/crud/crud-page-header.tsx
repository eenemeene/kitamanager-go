'use client';

import { useTranslations } from 'next-intl';
import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';

export interface CrudPageHeaderProps {
  /** i18n key for the title, or a string to display directly */
  title: string;
  /** i18n key for an optional one-line description shown under the title */
  description?: string;
  /** Handler for the "New" button click */
  onNew: () => void;
  /** i18n key for the "New" button text */
  newButtonText: string;
  /** Whether to hide the "New" button */
  hideNewButton?: boolean;
  /** Whether the "New" button is disabled */
  newButtonDisabled?: boolean;
  /** Extra elements rendered before the "New" button */
  children?: React.ReactNode;
}

/**
 * Reusable page header component for CRUD pages.
 * Displays a title, optional description, and a "New" button.
 */
export function CrudPageHeader({
  title,
  description,
  onNew,
  newButtonText,
  hideNewButton = false,
  newButtonDisabled = false,
  children,
}: CrudPageHeaderProps) {
  const t = useTranslations();

  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0">
        <h1 className="truncate text-2xl font-bold tracking-tight md:text-3xl">
          {title.includes('.') ? t(title) : title}
        </h1>
        {description && (
          <p className="text-muted-foreground mt-1 max-w-3xl text-sm">
            {/* Translation keys are dotted with no spaces; raw text always
                contains a space. Descriptions naturally contain periods,
                so we can't reuse the title's dot-only heuristic here. */}
            {description.includes(' ') ? description : t(description)}
          </p>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {children}
        {!hideNewButton && (
          <Button onClick={onNew} disabled={newButtonDisabled}>
            <Plus className="mr-2 h-4 w-4" />
            {newButtonText.includes('.') ? t(newButtonText) : newButtonText}
          </Button>
        )}
      </div>
    </div>
  );
}
