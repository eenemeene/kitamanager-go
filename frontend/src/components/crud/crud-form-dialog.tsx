'use client';

import type { FormEvent, ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';

export interface CrudFormDialogProps {
  /** Whether the dialog is open */
  open: boolean;
  /** Callback when open state changes */
  onOpenChange: (open: boolean) => void;
  /** Whether in edit mode (true) or create mode (false) */
  isEditing: boolean;
  /** i18n prefix for title keys (e.g., 'costs' → 'costs.edit' / 'costs.create') */
  translationPrefix: string;
  /** Form submit handler */
  onSubmit: (e: FormEvent<HTMLFormElement>) => void;
  /** Whether save is disabled (e.g., mutation in progress) */
  isSaving?: boolean;
  /** Form field content */
  children: ReactNode;
}

/**
 * Reusable CRUD form dialog with standard title, cancel/save footer.
 */
export function CrudFormDialog({
  open,
  onOpenChange,
  isEditing,
  translationPrefix,
  onSubmit,
  isSaving = false,
  children,
}: CrudFormDialogProps) {
  const t = useTranslations();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEditing ? t(`${translationPrefix}.edit`) : t(`${translationPrefix}.create`)}
          </DialogTitle>
        </DialogHeader>
        {/* The form is the flex column: its fields scroll, its actions do not.
            Seven pages render their dialogs through this component, so they all
            get a footer that stays put and never covers a field. */}
        <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
          <DialogBody className="space-y-4">{children}</DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={isSaving}>
              {t('common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
