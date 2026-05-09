'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Trash2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';

import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { useToast } from '@/lib/hooks/use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';
import { useCurrentRole, hasMinimumRole } from '@/hooks/use-current-role';
import type { Child, ChildVoucher } from '@/lib/api/types';

// Berlin Kita-Gutschein format: GB-DDDDDDDDDDD-NN. The same regex shape
// the backend's `voucher` Gin binding enforces — duplicated here for
// fast feedback before a network round-trip. Server stays the truth.
const VOUCHER_FORMAT = /^GB-\d{11}-\d{2}$/;
const VOUCHER_PLACEHOLDER = 'GB-12345678901-02';

export interface VouchersDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgId: number;
  child: Child | null;
}

export function VouchersDialog({ open, onOpenChange, orgId, child }: VouchersDialogProps) {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const role = useCurrentRole();
  // Add + remove are gated server-side via ResourceGovernmentFundingBills
  // (manager + above). Hiding the controls when the user lacks the role
  // is a UX-only convenience — a member who finds the buttons via dev
  // tools still 403s at the API.
  const canWrite = hasMinimumRole(role, 'manager');

  const [newNumber, setNewNumber] = useState('');
  const [formatError, setFormatError] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<ChildVoucher | null>(null);

  const childId = child?.id ?? 0;
  const enabled = open && childId > 0;

  const { data: vouchers, isLoading } = useQuery({
    queryKey: queryKeys.children.vouchers(orgId, childId),
    queryFn: () => apiClient.getChildVouchers(orgId, childId),
    enabled,
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.children.vouchers(orgId, childId) });
    // Dashboard "Children without vouchers" widget reflects voucher
    // assignment state — keep it in sync after add/remove.
    queryClient.invalidateQueries({ queryKey: queryKeys.children.withoutVouchers(orgId) });
  };

  const addMutation = useMutation({
    mutationFn: (voucherNumber: string) =>
      apiClient.assignChildVoucher(orgId, childId, voucherNumber),
    onSuccess: () => {
      toast({ title: t('vouchers.addSuccess') });
      setNewNumber('');
      setFormatError(null);
      invalidate();
    },
    onError: (error) => {
      showErrorToast(t('common.error'), error, t('vouchers.addError'));
    },
  });

  const removeMutation = useMutation({
    mutationFn: (voucherId: number) => apiClient.removeChildVoucher(orgId, childId, voucherId),
    onSuccess: () => {
      toast({ title: t('vouchers.removeSuccess') });
      setPendingDelete(null);
      invalidate();
    },
    onError: (error) => {
      showErrorToast(t('common.error'), error, t('vouchers.removeError'));
      setPendingDelete(null);
    },
  });

  const handleAdd = () => {
    const trimmed = newNumber.trim();
    if (!VOUCHER_FORMAT.test(trimmed)) {
      setFormatError(t('vouchers.formatHint'));
      return;
    }
    setFormatError(null);
    addMutation.mutate(trimmed);
  };

  const handleClose = (next: boolean) => {
    if (!next) {
      setNewNumber('');
      setFormatError(null);
      setPendingDelete(null);
    }
    onOpenChange(next);
  };

  return (
    <>
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t('vouchers.dialogTitle')}</DialogTitle>
            {child && (
              <DialogDescription>
                {child.first_name} {child.last_name}
              </DialogDescription>
            )}
          </DialogHeader>

          <div className="space-y-4">
            {/* Voucher list */}
            <div className="space-y-2">
              <Label>{t('vouchers.assignedLabel')}</Label>
              {isLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-11 w-full" />
                  <Skeleton className="h-11 w-full" />
                </div>
              ) : !vouchers || vouchers.length === 0 ? (
                <p className="text-muted-foreground text-sm">{t('vouchers.noneAssigned')}</p>
              ) : (
                <ul className="space-y-2" data-testid="voucher-list">
                  {vouchers.map((v) => (
                    <li
                      key={v.id}
                      className="flex items-center justify-between rounded-md border px-3 py-2"
                    >
                      <span className="font-mono text-sm">{v.voucher_number}</span>
                      {canWrite && (
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={t('vouchers.removeAction', { number: v.voucher_number })}
                          onClick={() => setPendingDelete(v)}
                          disabled={removeMutation.isPending}
                        >
                          <Trash2 className="text-destructive h-4 w-4" />
                        </Button>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </div>

            {/* Add input — managers + admins only */}
            {canWrite && (
              <div className="space-y-2">
                <Label htmlFor="voucher-add-input">{t('vouchers.addLabel')}</Label>
                <div className="flex gap-2">
                  <Input
                    id="voucher-add-input"
                    placeholder={VOUCHER_PLACEHOLDER}
                    value={newNumber}
                    onChange={(e) => {
                      setNewNumber(e.target.value);
                      if (formatError) setFormatError(null);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        handleAdd();
                      }
                    }}
                    disabled={addMutation.isPending}
                    autoComplete="off"
                    aria-invalid={formatError !== null}
                    aria-describedby="voucher-format-hint"
                  />
                  <Button
                    type="button"
                    onClick={handleAdd}
                    disabled={addMutation.isPending || newNumber.trim() === ''}
                  >
                    {t('vouchers.addAction')}
                  </Button>
                </div>
                <p id="voucher-format-hint" className="text-muted-foreground text-xs">
                  {formatError ?? t('vouchers.formatHint')}
                </p>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => handleClose(false)}>
              {t('common.close')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Hard-delete is irreversible; a confirmation dialog is the
          codebase-wide convention for destructive actions. */}
      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(next) => {
          if (!next) setPendingDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('vouchers.removeConfirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('vouchers.removeConfirmBody', { number: pendingDelete?.voucher_number ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => pendingDelete && removeMutation.mutate(pendingDelete.id)}
            >
              {t('common.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
