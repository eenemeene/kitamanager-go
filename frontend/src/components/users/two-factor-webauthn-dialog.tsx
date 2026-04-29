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
import { useEnrolWebAuthn, useActivateFactor } from '@/lib/hooks/use-factors';
import { factorEnrolSchema, type FactorEnrolFormData } from '@/lib/schemas/auth';
import type { BackupCodesPayload, WebAuthnEnrollmentPayload } from '@/lib/api/types';
import {
  decodeCreationOptions,
  encodeRegistrationResponse,
  isWebAuthnSupported,
} from '@/lib/utils/webauthn';

// Internal step of the WebAuthn enrol dialog. The ceremony is:
//   password → browser-prompt (navigator.credentials.create) → activate
// We keep both states here so the caller only has to know open/closed.
type EnrolStep = 'password' | 'prompting';

interface TwoFactorWebAuthnDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete: (payload: BackupCodesPayload | null) => void;
}

// TwoFactorWebAuthnDialog walks the user through enrolling a
// security key or platform passkey. State machine:
//
//   password  ─submit──▶  prompting (server issues challenge, browser prompts)
//                    ▲                 │
//                    │                 └─success──▶  onComplete(backup codes)
//                    │                 └─NotAllowed / cancel──▶ inline error, stay
//                    └─401 wrong password──▶ inline error, stay in password step
//
// Backup codes surfaced by the activate response are handed back to
// the parent card (which opens the backup-codes display dialog) —
// first-primary-factor activation on an account without any existing
// MFA triggers the same auto-create as TOTP enrolment.
export function TwoFactorWebAuthnDialog({
  open,
  onOpenChange,
  onComplete,
}: TwoFactorWebAuthnDialogProps) {
  const t = useTranslations('settings.twoFactor.webauthnDialog');
  const tParent = useTranslations('settings.twoFactor');
  const tCommon = useTranslations('common');
  const { toast } = useToast();

  const [step, setStep] = useState<EnrolStep>('password');
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [promptError, setPromptError] = useState<string | null>(null);

  const pwForm = useForm<FactorEnrolFormData>({
    resolver: zodResolver(factorEnrolSchema),
    mode: 'onChange',
    defaultValues: { password: '' },
  });

  const enrol = useEnrolWebAuthn();
  const activate = useActivateFactor();

  // Reset on close so a subsequent open starts fresh.
  useEffect(() => {
    if (!open) {
      setStep('password');
      setPasswordError(null);
      setPromptError(null);
      pwForm.reset({ password: '' });
      enrol.reset();
      activate.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const supported = isWebAuthnSupported();

  const runCeremony = async (password: string) => {
    setPromptError(null);
    // Step 1: server creates pending factor + issues challenge.
    const enrolResp = await enrol.mutateAsync({ password });
    const payload = enrolResp.enrollment as WebAuthnEnrollmentPayload | undefined;
    if (!payload?.creation_options) {
      throw new Error('server did not return creation_options');
    }
    // Step 2: hand options to the browser. This is where the user
    // touches their security key / uses Touch ID / etc.
    const opts = decodeCreationOptions(
      payload.creation_options as unknown as Parameters<typeof decodeCreationOptions>[0]
    );
    const cred = (await navigator.credentials.create({
      publicKey: opts,
    })) as PublicKeyCredential | null;
    if (!cred) {
      throw new Error('no credential returned');
    }
    // Step 3: send the attestation back to the server; it verifies
    // and stores. On first-primary activation the response carries
    // backup codes.
    const encoded = encodeRegistrationResponse(cred);
    const activateResp = await activate.mutateAsync({
      factorId: enrolResp.id,
      webauthnResponse: encoded,
    });
    toast({ title: tParent('successEnabled') });
    onComplete(activateResp.backup_codes ?? null);
  };

  const onPasswordSubmit = pwForm.handleSubmit(async (data) => {
    setPasswordError(null);
    setStep('prompting');
    try {
      await runCeremony(data.password);
    } catch (err) {
      const axiosErr = err as AxiosError;
      const status = axiosErr.response?.status;
      // 401 means the step-up password was wrong — drop back to the
      // password step so the user can re-enter it.
      if (status === 401) {
        setStep('password');
        setPasswordError(t('wrongPassword'));
        pwForm.setValue('password', '');
        return;
      }
      // Everything else (server 409, browser NotAllowedError, etc.)
      // keeps the user on the prompting step so the inline error and
      // the Retry button are visible. Reverting to password here
      // would silently eat the error since promptError is only
      // rendered on the prompting step.
      if (status === 409) {
        setPromptError(t('alreadyRegistered'));
        return;
      }
      const name = (err as { name?: string }).name;
      if (name === 'NotAllowedError' || name === 'AbortError') {
        setPromptError(t('userCancelled'));
        return;
      }
      if (name === 'InvalidStateError') {
        setPromptError(t('alreadyRegistered'));
        return;
      }
      // Unknown / network failure — surface the toast and drop back
      // so the user can retry from the top.
      setStep('password');
      toast({ title: tCommon('error'), variant: 'destructive' });
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>
            {step === 'password' ? t('description') : t('prompting')}
          </DialogDescription>
        </DialogHeader>

        {!supported ? (
          <p role="alert" className="text-destructive text-sm">
            {t('unsupported')}
          </p>
        ) : step === 'password' ? (
          <form onSubmit={onPasswordSubmit} className="space-y-4" noValidate>
            <div className="space-y-1">
              <Label htmlFor="wa-enrol-password">{t('passwordLabel')}</Label>
              <Input
                id="wa-enrol-password"
                type="password"
                autoComplete="current-password"
                aria-invalid={!!passwordError}
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
          <div className="space-y-4">
            <p className="text-muted-foreground text-sm">{t('prompting')}</p>
            {promptError && (
              <p role="alert" className="text-destructive text-sm">
                {promptError}
              </p>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t('cancel')}
              </Button>
              {promptError && (
                <Button
                  type="button"
                  onClick={() => runCeremony(pwForm.getValues('password'))}
                  disabled={activate.isPending}
                >
                  {t('retry')}
                </Button>
              )}
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
