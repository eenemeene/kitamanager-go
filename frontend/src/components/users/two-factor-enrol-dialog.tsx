'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { AxiosError } from 'axios';
import { QRCodeSVG } from 'qrcode.react';

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
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp';
import { useToast } from '@/lib/hooks/use-toast';
import { useEnrolTotp, useActivateFactor } from '@/lib/hooks/use-factors';
import {
  factorActivateSchema,
  factorEnrolSchema,
  type FactorActivateFormData,
  type FactorEnrolFormData,
} from '@/lib/schemas/auth';
import type { BackupCodesPayload, FactorResponse } from '@/lib/api/types';

// Internal step of the enrol dialog. The parent card only knows
// about "open" or "closed" — the two-step UX lives here.
type EnrolStep = 'password' | 'scan';

interface TwoFactorEnrolDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  // onComplete fires when the user has successfully activated their
  // factor AND the backend returned a fresh set of backup codes. The
  // parent hands those to the BackupCodesDialog for one-shot display.
  onComplete: (payload: BackupCodesPayload) => void;
}

export function TwoFactorEnrolDialog({
  open,
  onOpenChange,
  onComplete,
}: TwoFactorEnrolDialogProps) {
  const t = useTranslations('settings.twoFactor.enrolDialog');
  const tParent = useTranslations('settings.twoFactor');
  const tCommon = useTranslations('common');
  const { toast } = useToast();

  const [step, setStep] = useState<EnrolStep>('password');
  // The enrolled factor row (id + secret + otpauth_uri) lives here
  // until activation completes. On cancel or close we forget it —
  // the server-side pending row is swept by the cleanup job.
  const [factor, setFactor] = useState<FactorResponse | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [codeError, setCodeError] = useState<string | null>(null);

  const pwForm = useForm<FactorEnrolFormData>({
    resolver: zodResolver(factorEnrolSchema),
    mode: 'onChange',
    defaultValues: { password: '' },
  });
  const codeForm = useForm<FactorActivateFormData>({
    resolver: zodResolver(factorActivateSchema),
    mode: 'onChange',
    defaultValues: { code: '' },
  });

  const enrol = useEnrolTotp();
  const activate = useActivateFactor();

  // Reset on close so a subsequent open starts fresh — critical for
  // the error-then-retry test (the same-render second attempt must
  // not carry over stale state).
  useEffect(() => {
    if (!open) {
      setStep('password');
      setFactor(null);
      setPasswordError(null);
      setCodeError(null);
      pwForm.reset({ password: '' });
      codeForm.reset({ code: '' });
      enrol.reset();
      activate.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const onPasswordSubmit = pwForm.handleSubmit((data) => {
    setPasswordError(null);
    enrol.mutate(
      { password: data.password },
      {
        onSuccess: (row) => {
          setFactor(row);
          setStep('scan');
        },
        onError: (err) => {
          const status = (err as AxiosError).response?.status;
          if (status === 401) {
            setPasswordError(t('wrongPassword'));
          } else {
            toast({ title: tCommon('error'), variant: 'destructive' });
          }
          pwForm.setValue('password', '');
        },
      }
    );
  });

  const onCodeSubmit = codeForm.handleSubmit((data) => {
    if (!factor) return;
    setCodeError(null);
    activate.mutate(
      { factorId: factor.id, code: data.code },
      {
        onSuccess: (res) => {
          if (res.backup_codes) {
            toast({ title: tParent('successEnabled') });
            onComplete(res.backup_codes);
          } else {
            // Defensive — backend always returns codes on first
            // primary activation, but if we somehow land without
            // them, at least close the dialog cleanly.
            toast({ title: tParent('successEnabled') });
            onOpenChange(false);
          }
        },
        onError: (err) => {
          const status = (err as AxiosError).response?.status;
          if (status === 429) {
            // Server destroyed the pending row. Bail: close dialog,
            // show banner. User has to restart.
            toast({ title: t('tooManyWrongCodes'), variant: 'destructive' });
            onOpenChange(false);
          } else if (status === 401) {
            setCodeError(t('wrongCode'));
            codeForm.setValue('code', '');
          } else {
            toast({ title: tCommon('error'), variant: 'destructive' });
          }
        },
      }
    );
  });

  const payload = factor?.enrollment;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>
            {step === 'password' ? t('description') : t('scanDescription')}
          </DialogDescription>
        </DialogHeader>

        {step === 'password' ? (
          <form onSubmit={onPasswordSubmit} className="space-y-4" noValidate>
            <div className="space-y-1">
              <Label htmlFor="enrol-password">{t('passwordLabel')}</Label>
              <Input
                id="enrol-password"
                type="password"
                autoComplete="current-password"
                aria-invalid={!!passwordError || !!pwForm.formState.errors.password}
                {...pwForm.register('password')}
              />
              {passwordError && (
                <p role="alert" className="text-destructive text-sm">
                  {passwordError}
                </p>
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t('cancel')}
              </Button>
              <Button type="submit" disabled={!pwForm.formState.isValid || enrol.isPending}>
                {t('continue')}
              </Button>
            </DialogFooter>
          </form>
        ) : (
          <form onSubmit={onCodeSubmit} className="space-y-4" noValidate>
            {payload && (
              <div className="flex flex-col items-center gap-3">
                <div className="rounded-md bg-white p-3">
                  <QRCodeSVG
                    value={payload.otpauth_uri}
                    size={192}
                    level="M"
                    includeMargin={false}
                  />
                </div>
                <p className="text-muted-foreground text-xs">
                  {t('secretHint')} <code className="font-mono break-all">{payload.secret}</code>
                </p>
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="enrol-code">{t('codeLabel')}</Label>
              <div className="flex justify-center">
                <InputOTP
                  maxLength={6}
                  id="enrol-code"
                  value={codeForm.watch('code')}
                  onChange={(v) => codeForm.setValue('code', v, { shouldValidate: true })}
                >
                  <InputOTPGroup>
                    <InputOTPSlot index={0} />
                    <InputOTPSlot index={1} />
                    <InputOTPSlot index={2} />
                    <InputOTPSlot index={3} />
                    <InputOTPSlot index={4} />
                    <InputOTPSlot index={5} />
                  </InputOTPGroup>
                </InputOTP>
              </div>
              {codeError && (
                <p role="alert" className="text-destructive text-center text-sm">
                  {codeError}
                </p>
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t('cancel')}
              </Button>
              <Button type="submit" disabled={!codeForm.formState.isValid || activate.isPending}>
                {t('activate')}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
