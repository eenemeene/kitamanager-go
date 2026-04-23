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
import { useDeleteFactor } from '@/lib/hooks/use-factors';
import { factorDisableSchema, type FactorDisableFormData } from '@/lib/schemas/auth';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  // The id of the primary factor to disable. Since disabling the
  // last primary also sweeps the backup_codes factor at the service
  // layer, targeting the TOTP factor is sufficient.
  factorId: number | undefined;
}

export function TwoFactorDisableDialog({ open, onOpenChange, factorId }: Props) {
  const t = useTranslations('settings.twoFactor.disableDialog');
  const tParent = useTranslations('settings.twoFactor');
  const tCommon = useTranslations('common');
  const { toast } = useToast();

  const [error, setError] = useState<string | null>(null);
  const form = useForm<FactorDisableFormData>({
    resolver: zodResolver(factorDisableSchema),
    mode: 'onChange',
    defaultValues: { password: '', code: '' },
  });
  const mutation = useDeleteFactor();

  useEffect(() => {
    if (!open) {
      setError(null);
      form.reset({ password: '', code: '' });
      mutation.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const onSubmit = form.handleSubmit((data) => {
    if (!factorId) return;
    setError(null);
    mutation.mutate(
      { factorId, password: data.password, code: data.code },
      {
        onSuccess: () => {
          toast({ title: tParent('successDisabled') });
          onOpenChange(false);
        },
        onError: (err) => {
          const status = (err as AxiosError).response?.status;
          if (status === 401 || status === 400) {
            setError(t('wrongPasswordOrCode'));
          } else {
            toast({ title: tCommon('error'), variant: 'destructive' });
          }
          form.setValue('password', '');
          form.setValue('code', '');
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
            <Label htmlFor="disable-password">{t('passwordLabel')}</Label>
            <Input
              id="disable-password"
              type="password"
              autoComplete="current-password"
              aria-invalid={!!error}
              {...form.register('password')}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="disable-code">{t('codeLabel')}</Label>
            <Input
              id="disable-code"
              type="text"
              inputMode="text"
              autoComplete="one-time-code"
              aria-invalid={!!error}
              {...form.register('code')}
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
              variant="destructive"
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
