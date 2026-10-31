'use client';

import { useState } from 'react';
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent autoFocusFirstField>
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>{t('description')}</DialogDescription>
        </DialogHeader>
        {/*
          The form is its own component so that the password, the code, the
          error and the mutation live *inside* DialogContent. Radix unmounts
          that subtree when the dialog closes, so closing resets them for free
          and reopening cannot show the previous attempt.

          This used to be an effect on the parent that reset the four by hand
          whenever `open` went false, which is the thing react-hooks 7.1's
          set-state-in-effect rule objects to -- and the effect existed only
          because the state sat one level too high, above the boundary that
          already has the lifetime we wanted. There is nothing to synchronize
          here, so there is nothing for an effect to do.
        */}
        <DisableForm onOpenChange={onOpenChange} factorId={factorId} />
      </DialogContent>
    </Dialog>
  );
}

function DisableForm({ onOpenChange, factorId }: Omit<Props, 'open'>) {
  const t = useTranslations('settings.twoFactor.disableDialog');
  const tParent = useTranslations('settings.twoFactor');
  const tCommon = useTranslations('common');
  const { toast } = useToast();

  const [error, setError] = useState<string | null>(null);
  const form = useForm<FactorDisableFormData>({
    resolver: zodResolver(factorDisableSchema),
    // Deliberately onChange, not the shared validationTiming. This form gates
    // its submit button on formState.isValid, and under `onTouched` that stays
    // false until the field is blurred — so someone who types the code and goes
    // straight for the button finds it disabled. Short form, fixed-length input,
    // no "told you are wrong too early" problem to solve here.
    mode: 'onChange',
    defaultValues: { password: '', code: '' },
  });
  const mutation = useDeleteFactor();

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
  );
}
