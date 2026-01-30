'use client';

import { useTranslations } from 'next-intl';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

export interface DeleteConfirmDialogProps {
  /** Whether the dialog is open */
  open: boolean;
  /** Callback when open state changes */
  onOpenChange: (open: boolean) => void;
  /** Callback when delete is confirmed */
  onConfirm: () => void;
  /** Whether the delete operation is in progress */
  isLoading?: boolean;
  /** Resource name for i18n (e.g., 'groups', 'organizations') */
  resourceName: string;
  /** Optional custom description. If not provided, uses `{resourceName}.deleteConfirm` */
  description?: string;
}

/**
 * Reusable delete confirmation dialog component.
 * Uses i18n keys based on the resourceName prop.
 */
export function DeleteConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  isLoading = false,
  resourceName,
  description,
}: DeleteConfirmDialogProps) {
  const t = useTranslations();

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('common.confirmDelete')}</AlertDialogTitle>
          <AlertDialogDescription>
            {description ?? t(`${resourceName}.deleteConfirm`)}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isLoading}>{t('common.cancel')}</AlertDialogCancel>
          <AlertDialogAction
            onClick={onConfirm}
            disabled={isLoading}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {t('common.delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
