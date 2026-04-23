'use client';

import { useTranslations } from 'next-intl';
import { ChangePasswordCard } from '@/components/users/change-password-card';
import { ActiveSessionsCard } from '@/components/users/active-sessions-card';
import { TwoFactorCard } from '@/components/users/two-factor-card';

export default function SettingsPage() {
  const t = useTranslations();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{t('nav.settings')}</h1>
      </div>
      <ChangePasswordCard />
      <TwoFactorCard />
      <ActiveSessionsCard />
    </div>
  );
}
