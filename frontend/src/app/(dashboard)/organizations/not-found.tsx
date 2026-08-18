'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { Building2 } from 'lucide-react';
import { Button } from '@/components/ui/button';

/**
 * Shown when the addressed organization does not exist, or belongs to someone
 * else. Deliberately inside the dashboard layout: the sidebar and header stay,
 * so the reader can go somewhere useful instead of reaching for the back button.
 *
 * It sits on `organizations/`, not on `organizations/[orgId]/`, because the
 * guard that triggers it is the `[orgId]` layout -- and a layout's own
 * not-found boundary renders *inside* that layout, which is the thing that
 * threw. Putting it here got the app's own page instead of Next's bare "404".
 */
export default function OrganizationNotFound() {
  const t = useTranslations();

  return (
    <div className="flex flex-col items-center justify-center gap-4 py-24 text-center">
      <Building2 className="text-muted-foreground h-10 w-10" aria-hidden="true" />
      <div className="space-y-1">
        <h1 className="text-xl font-semibold">{t('organizations.notFoundTitle')}</h1>
        <p className="text-muted-foreground max-w-md text-sm">
          {t('organizations.notFoundDescription')}
        </p>
      </div>
      <Button asChild>
        <Link href="/organizations">{t('organizations.notFoundAction')}</Link>
      </Button>
    </div>
  );
}
