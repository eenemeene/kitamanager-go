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

export function MfaVerifyForm({ pendingToken, factors, onRestart, onSuccess }: Props) {
  const t = useTranslations('auth.mfa');
  const tSettings = useTranslations('settings.twoFactor');
  const hydrate = useAuthStore((s) => s.hydrateAfterAuth);

  // Default selection: the first factor in the list. The backend's
  // FindActiveByUserID already orders primaries first, so this is
  // usually the most recently used TOTP.
  const [selectedFactorId, setSelectedFactorId] = useState<number>(factors[0]?.id ?? 0);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const { register, handleSubmit, setValue, formState } = useForm<MfaVerifyFormData>({
    resolver: zodResolver(mfaVerifySchema),
    mode: 'onChange',
    defaultValues: { code: '' },
  });

  const factorLabel = (f: LoginFactorDescriptor) => {
    if (f.type === 'totp') return f.label || tSettings('factorTypeTotp');
    return tSettings('factorTypeBackupCodes');
  };

  const onSubmit = handleSubmit(async (data) => {
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
      const status = (err as AxiosError).response?.status;
      if (status === 429) {
        onRestart('too_many');
      } else if (status === 401) {
        // Could be either "wrong code" or "pending expired" —
        // indistinguishable from this side. Keep the user on the
        // form for "wrong code" (the common case) and let them
        // retry; if the pending is truly expired, subsequent
        // attempts will also 401 and the user can press Back.
        setError(t('wrongCode'));
        setValue('code', '');
      } else {
        setError(t('wrongCode'));
        setValue('code', '');
      }
    } finally {
      setSubmitting(false);
    }
  });

  return (
    <form data-testid="mfa-verify-form" onSubmit={onSubmit} className="space-y-4" noValidate>
      <div className="space-y-1">
        <p className="text-sm font-medium">{t('title')}</p>
        <p className="text-muted-foreground text-sm">{t('description')}</p>
      </div>

      {factors.length > 1 && (
        <div className="space-y-1">
          <Label htmlFor="factor-picker">{t('factorPickerLabel')}</Label>
          <Select
            value={String(selectedFactorId)}
            onValueChange={(v) => setSelectedFactorId(Number(v))}
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
        <Button
          type="submit"
          disabled={!formState.isValid || submitting || !selectedFactorId}
          className="flex-1"
        >
          {t('submit')}
        </Button>
      </div>
    </form>
  );
}
