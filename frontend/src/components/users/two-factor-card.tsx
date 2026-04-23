'use client';

import { useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { formatDistanceToNow } from 'date-fns';
import { de, enUS } from 'date-fns/locale';
import { Shield, ShieldCheck } from 'lucide-react';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useFactors } from '@/lib/hooks/use-factors';
import type { FactorResponse, BackupCodesPayload } from '@/lib/api/types';

import { TwoFactorEnrolDialog } from './two-factor-enrol-dialog';
import { TwoFactorBackupCodesDialog } from './two-factor-backup-codes-dialog';
import { TwoFactorRegenerateDialog } from './two-factor-regenerate-dialog';
import { TwoFactorDisableDialog } from './two-factor-disable-dialog';
import { TwoFactorWebAuthnDialog } from './two-factor-webauthn-dialog';
import { isWebAuthnSupported } from '@/lib/utils/webauthn';

// TwoFactorCard is the Settings entry for MFA enrolment. The component
// owns a small state machine:
//
//   loading ─► (disabled | enabled)
//   disabled ──(click Enable)──► enrolling (EnrolDialog) ──(activate)──► saving-backup-codes (BackupCodesDialog) ──(ack)──► enabled
//   enabled ──(click Regenerate)──► regenerating (RegenerateDialog) ──(done)──► saving-backup-codes
//   enabled ──(click Disable)──► disabling (DisableDialog) ──(done)──► disabled
//
// Backup codes are held ONLY in this component's state — never in
// localStorage, Zustand, or the react-query cache — and dropped the
// moment the user acknowledges them. The list query only returns
// `backup_codes_remaining`, never the codes themselves.
export function TwoFactorCard() {
  const t = useTranslations('settings.twoFactor');
  const tCommon = useTranslations('common');
  const locale = useLocale();
  const dateLocale = locale === 'de' ? de : enUS;

  // Transient one-shot state. After the user clicks "Done" in the
  // backup-codes dialog, this is cleared and can't be recovered.
  const [backupPayload, setBackupPayload] = useState<BackupCodesPayload | null>(null);

  // Dialog visibility. Keeping these as independent booleans lets the
  // state machine make explicit transitions (e.g. close Enrol, then
  // open BackupCodes) without the dialogs fighting over focus.
  const [enrolOpen, setEnrolOpen] = useState(false);
  const [webAuthnOpen, setWebAuthnOpen] = useState(false);
  const [regenerateOpen, setRegenerateOpen] = useState(false);
  const [disableOpen, setDisableOpen] = useState(false);

  // Factor lookups. TOTP and webauthn are both "primary" factors;
  // the first primary drives the regenerate/disable actions (future:
  // per-row action buttons when the user has multiple primaries).
  const query = useFactors();
  const factors = query.data?.factors ?? [];
  const totp = factors.find((f) => f.type === 'totp');
  const webAuthnFactors = factors.filter((f) => f.type === 'webauthn');
  const backupCodes = factors.find((f) => f.type === 'backup_codes');
  const hasMfa = Boolean(totp) || webAuthnFactors.length > 0;

  const formatRelative = (iso: string | undefined) =>
    iso ? formatDistanceToNow(new Date(iso), { addSuffix: true, locale: dateLocale }) : null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          {hasMfa ? (
            <ShieldCheck className="h-5 w-5" aria-hidden />
          ) : (
            <Shield className="h-5 w-5" aria-hidden />
          )}
          {t('title')}
        </CardTitle>
        <CardDescription>{t('description')}</CardDescription>
      </CardHeader>
      <CardContent>
        {query.isPending ? (
          <p className="text-muted-foreground text-sm">{t('loading')}</p>
        ) : query.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t('loadError')}
          </p>
        ) : !hasMfa ? (
          <div className="flex flex-col gap-3">
            <p className="text-muted-foreground text-sm">{t('statusDisabled')}</p>
            <div className="flex flex-wrap gap-2">
              <Button onClick={() => setEnrolOpen(true)}>{t('enableButton')}</Button>
              {isWebAuthnSupported() && (
                <Button variant="outline" onClick={() => setWebAuthnOpen(true)}>
                  {t('addSecurityKeyButton')}
                </Button>
              )}
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Badge variant="default">{t('statusEnabled')}</Badge>
            </div>
            <ul className="space-y-2">
              {factors.map((f) => (
                <FactorListRow
                  key={f.id}
                  factor={f}
                  label={labelFor(t, f)}
                  secondary={secondaryFor(t, tCommon, f, formatRelative)}
                />
              ))}
            </ul>
            <div className="flex flex-wrap gap-2">
              {isWebAuthnSupported() && (
                <Button variant="outline" onClick={() => setWebAuthnOpen(true)}>
                  {t('addSecurityKeyButton')}
                </Button>
              )}
              <Button variant="outline" onClick={() => setRegenerateOpen(true)}>
                {t('regenerateButton')}
              </Button>
              <Button variant="destructive" onClick={() => setDisableOpen(true)}>
                {t('disableButton')}
              </Button>
            </div>
          </div>
        )}
      </CardContent>

      <TwoFactorEnrolDialog
        open={enrolOpen}
        onOpenChange={setEnrolOpen}
        onComplete={(payload) => {
          setEnrolOpen(false);
          setBackupPayload(payload);
        }}
      />

      <TwoFactorWebAuthnDialog
        open={webAuthnOpen}
        onOpenChange={setWebAuthnOpen}
        onComplete={(payload) => {
          setWebAuthnOpen(false);
          // payload may be null — first-primary activation issues
          // backup codes, subsequent webauthn enrolments don't.
          if (payload) setBackupPayload(payload);
        }}
      />

      <TwoFactorRegenerateDialog
        open={regenerateOpen}
        onOpenChange={setRegenerateOpen}
        factorId={backupCodes?.id}
        onComplete={(payload) => {
          setRegenerateOpen(false);
          setBackupPayload(payload);
        }}
      />

      <TwoFactorDisableDialog
        open={disableOpen}
        onOpenChange={setDisableOpen}
        factorId={totp?.id}
      />

      <TwoFactorBackupCodesDialog payload={backupPayload} onClose={() => setBackupPayload(null)} />
    </Card>
  );
}

function FactorListRow({
  factor,
  label,
  secondary,
}: {
  factor: FactorResponse;
  label: string;
  secondary: string | null;
}) {
  return (
    <li
      data-testid={`factor-row-${factor.type}`}
      className="flex items-center justify-between gap-3 text-sm"
    >
      <div className="min-w-0">
        <p className="font-medium">{label}</p>
        {secondary && <p className="text-muted-foreground truncate">{secondary}</p>}
      </div>
    </li>
  );
}

function labelFor(t: ReturnType<typeof useTranslations>, f: FactorResponse): string {
  if (f.type === 'totp') return f.label || t('factorTypeTotp');
  if (f.type === 'webauthn') return f.label || t('factorTypeWebAuthn');
  return t('factorTypeBackupCodes');
}

function secondaryFor(
  t: ReturnType<typeof useTranslations>,
  _tCommon: ReturnType<typeof useTranslations>,
  f: FactorResponse,
  formatRelative: (iso: string | undefined) => string | null
): string | null {
  if (f.type === 'backup_codes') {
    const remaining = f.backup_codes_remaining ?? 0;
    return t('backupCodesRemaining', { count: remaining });
  }
  const when = formatRelative(f.last_used_at);
  return when ? t('lastUsed', { when }) : t('lastUsedNever');
}
