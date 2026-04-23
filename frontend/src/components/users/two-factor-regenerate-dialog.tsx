'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { AxiosError } from 'axios';

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useToast } from '@/lib/hooks/use-toast';
import { useRegenerateBackupCodes } from '@/lib/hooks/use-factors';
import { factorPasswordStepUpSchema, type FactorPasswordStepUpFormData } from '@/lib/schemas/auth';
import type { BackupCodesPayload } from '@/lib/api/types';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  // The id of the backup_codes factor to regenerate against. Null
  // means the card is still loading the factor list; the dialog
  // should not open in that case, but we guard against it by
  // disabling the submit button.
  factorId: number | undefined;
  onComplete: (payload: BackupCodesPayload) => void;
}

export function TwoFactorRegenerateDialog({ open, onOpenChange, factorId, onComplete }: Props) {
  const t = useTranslations('settings.twoFactor.regenerateDialog');
  const tParent = useTranslations('settings.twoFactor');
  const tCommon = useTranslations('common');
  const { toast } = useToast();

  const [error, setError] = useState<string | null>(null);
  const form = useForm<FactorPasswordStepUpFormData>({
    resolver: zodResolver(factorPasswordStepUpSchema),
    mode: 'onChange',
    defaultValues: { password: '' },
  });
  const mutation = useRegenerateBackupCodes();

  useEffect(() => {
    if (!open) {
      setError(null);
      form.reset({ password: '' });
      mutation.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const onSubmit = form.handleSubmit((data) => {
    if (!factorId) return;
    setError(null);
    mutation.mutate(
      { factorId, password: data.password },
      {
        onSuccess: (payload) => {
          toast({ title: tParent('successRegenerated') });
          onComplete(payload);
        },
        onError: (err) => {
          const status = (err as AxiosError).response?.status;
          if (status === 401) {
            setError(t('wrongPassword'));
          } else {
            toast({ title: tCommon('error'), variant: 'destructive' });
          }
          form.setValue('password', '');
        },
      }
    );
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>{t('description')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4" noValidate>
          <div className="space-y-1">
            <Label htmlFor="regen-password">{t('passwordLabel')}</Label>
            <Input
              id="regen-password"
              type="password"
              autoComplete="current-password"
              aria-invalid={!!error}
              {...form.register('password')}
            />
            {error && (
              <p role="alert" className="text-destructive text-sm">
                {error}
              </p>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('cancel')}
            </Button>
            <Button
              type="submit"
              disabled={!form.formState.isValid || mutation.isPending || !factorId}
            >
              {t('submit')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
