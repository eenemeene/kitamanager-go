'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { AxiosError } from 'axios';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api/client';
import { useAuthStore } from '@/stores/auth-store';
import { mfaVerifySchema, type MfaVerifyFormData } from '@/lib/schemas/auth';
import type { LoginFactorDescriptor } from '@/lib/api/types';
import { decodeRequestOptions, encodeAssertionResponse } from '@/lib/utils/webauthn';

interface Props {
  pendingToken: string;
  factors: LoginFactorDescriptor[];
  // Called with a "back to password" reason so the page can decide
  // whether to surface a banner. `too_many` means the pending row
  // was destroyed by rate limit and the user has to restart; `user`
  // is just the back button.
  onRestart: (reason: 'user' | 'too_many' | 'expired') => void;
  onSuccess: () => void;
}

// MfaVerifyForm is the second-step of two-step login. It dispatches
// internally on the currently-picked factor's type:
//
//   totp / backup_codes  →  6-digit code input + submit
//   webauthn             →  "Use security key" button → navigator.credentials.get()
//
// Both branches end in the same /auth/mfa/verify call (body shape
// differs by the code vs webauthn_response field).
export function MfaVerifyForm({ pendingToken, factors, onRestart, onSuccess }: Props) {
  const t = useTranslations('auth.mfa');
  const tSettings = useTranslations('settings.twoFactor');
  const hydrate = useAuthStore((s) => s.hydrateAfterAuth);

  const [selectedFactorId, setSelectedFactorId] = useState<number>(factors[0]?.id ?? 0);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const selected = factors.find((f) => f.id === selectedFactorId);
  const isWebAuthn = selected?.type === 'webauthn';

  const { register, handleSubmit, setValue, formState } = useForm<MfaVerifyFormData>({
    resolver: zodResolver(mfaVerifySchema),
    // Deliberately onChange, not the shared validationTiming. This form gates
    // its submit button on formState.isValid, and under `onTouched` that stays
    // false until the field is blurred — so someone who types the code and goes
    // straight for the button finds it disabled. Short form, fixed-length input,
    // no "told you are wrong too early" problem to solve here.
    mode: 'onChange',
    defaultValues: { code: '' },
  });

  const factorLabel = (f: LoginFactorDescriptor) => {
    if (f.type === 'totp') return f.label || tSettings('factorTypeTotp');
    if (f.type === 'webauthn') return f.label || tSettings('factorTypeWebAuthn');
    return tSettings('factorTypeBackupCodes');
  };

  const classifyErrorAndRestart = (err: unknown): boolean => {
    const status = (err as AxiosError).response?.status;
    if (status === 429) {
      onRestart('too_many');
      return true;
    }
    return false;
  };

  const onCodeSubmit = handleSubmit(async (data) => {
    setError(null);
    setSubmitting(true);
    try {
      await apiClient.verifyMfa({
        pending_token: pendingToken,
        factor_id: selectedFactorId,
        code: data.code,
      });
      await hydrate();
      onSuccess();
    } catch (err) {
      if (classifyErrorAndRestart(err)) return;
      setError(t('wrongCode'));
      setValue('code', '');
    } finally {
      setSubmitting(false);
    }
  });

  const onWebAuthnSubmit = async () => {
    setError(null);
    setSubmitting(true);
    try {
      // Step 1: fetch challenge from the server; stored on the
      // pending row so the verify call can look it up.
      const challenge = await apiClient.beginMfaChallenge({
        pending_token: pendingToken,
        factor_id: selectedFactorId,
      });
      const opts = decodeRequestOptions(
        challenge.request_options as unknown as Parameters<typeof decodeRequestOptions>[0]
      );
      // Step 2: browser prompts the user — touch security key / Face ID / etc.
      const cred = (await navigator.credentials.get({
        publicKey: opts,
      })) as PublicKeyCredential | null;
      if (!cred) throw new Error('no credential');
      const encoded = encodeAssertionResponse(cred);
      // Step 3: server verifies assertion and issues the session.
      await apiClient.verifyMfa({
        pending_token: pendingToken,
        factor_id: selectedFactorId,
        webauthn_response: encoded as unknown as Record<string, never>,
      });
      await hydrate();
      onSuccess();
    } catch (err) {
      if (classifyErrorAndRestart(err)) return;
      const name = (err as { name?: string }).name;
      if (name === 'NotAllowedError' || name === 'AbortError') {
        setError(t('webauthnCancelled'));
      } else {
        setError(t('webauthnFailed'));
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form
      data-testid="mfa-verify-form"
      onSubmit={isWebAuthn ? (e) => e.preventDefault() : onCodeSubmit}
      className="space-y-4"
      noValidate
    >
      <div className="space-y-1">
        <p className="text-sm font-medium">{t('title')}</p>
        <p className="text-muted-foreground text-sm">{t('description')}</p>
      </div>

      {factors.length > 1 && (
        <div className="space-y-1">
          <Label htmlFor="factor-picker">{t('factorPickerLabel')}</Label>
          <Select
            value={String(selectedFactorId)}
            onValueChange={(v) => {
              setSelectedFactorId(Number(v));
              setError(null);
            }}
          >
            <SelectTrigger id="factor-picker">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {factors.map((f) => (
                <SelectItem key={f.id} value={String(f.id)}>
                  {factorLabel(f)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      {isWebAuthn ? (
        <div className="space-y-2">
          <p className="text-muted-foreground text-sm">{t('webauthnPrompt')}</p>
          {error && (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          )}
        </div>
      ) : (
        <div className="space-y-1">
          <Label htmlFor="mfa-code">{t('codeLabel')}</Label>
          <Input
            id="mfa-code"
            type="text"
            inputMode="text"
            autoComplete="one-time-code"
            autoFocus
            aria-invalid={!!error}
            aria-live="polite"
            {...register('code')}
            disabled={submitting}
          />
          {error && (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          )}
        </div>
      )}

      <p className="text-muted-foreground text-xs">{t('lostAccessHint')}</p>

      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={() => onRestart('user')}
          disabled={submitting}
        >
          {t('back')}
        </Button>
        {isWebAuthn ? (
          <Button
            type="button"
            onClick={onWebAuthnSubmit}
            disabled={submitting || !selectedFactorId}
            className="flex-1"
          >
            {t('webauthnSubmit')}
          </Button>
        ) : (
          <Button
            type="submit"
            disabled={!formState.isValid || submitting || !selectedFactorId}
            className="flex-1"
          >
            {t('submit')}
          </Button>
        )}
      </div>
    </form>
  );
}
