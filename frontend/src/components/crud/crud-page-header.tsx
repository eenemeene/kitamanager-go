'use client';

import { useTranslations } from 'next-intl';
import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';

export interface CrudPageHeaderProps {
  /** i18n key for the title, or a string to display directly */
  title: string;
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
 * Displays a title and a "New" button.
 */
export function CrudPageHeader({
  title,
  onNew,
  newButtonText,
  hideNewButton = false,
  newButtonDisabled = false,
  children,
}: CrudPageHeaderProps) {
  const t = useTranslations();

  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <h1 className="truncate text-2xl font-bold tracking-tight md:text-3xl">
          {title.includes('.') ? t(title) : title}
        </h1>
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
