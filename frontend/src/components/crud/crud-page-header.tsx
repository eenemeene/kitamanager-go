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
}: CrudPageHeaderProps) {
  const t = useTranslations();

  return (
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          {title.includes('.') ? t(title) : title}
        </h1>
      </div>
      {!hideNewButton && (
        <Button onClick={onNew} disabled={newButtonDisabled}>
          <Plus className="mr-2 h-4 w-4" />
          {newButtonText.includes('.') ? t(newButtonText) : newButtonText}
        </Button>
      )}
    </div>
  );
}
