'use client';

import { useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { formatDistanceToNow } from 'date-fns';
import { de, enUS } from 'date-fns/locale';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient } from '@/lib/api/client';
import type { UserSession } from '@/lib/api/types';

const SESSIONS_QUERY_KEY = ['me', 'sessions'] as const;

export function ActiveSessionsCard() {
  const t = useTranslations('settings.sessions');
  const tCommon = useTranslations('common');
  const locale = useLocale();
  const dateLocale = locale === 'de' ? de : enUS;
  const { toast } = useToast();
  const queryClient = useQueryClient();

  // Session pending revocation confirmation. `null` means the dialog is
  // closed. Held here so the dialog can render outside the row-level
  // `Button` that triggered it — keeps the table layout clean.
  const [pendingRevoke, setPendingRevoke] = useState<UserSession | null>(null);

  const query = useQuery({
    queryKey: SESSIONS_QUERY_KEY,
    queryFn: () => apiClient.getSessions(),
  });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => apiClient.revokeSession(id),
    onSuccess: () => {
      toast({ title: t('revokeSuccessToast') });
      setPendingRevoke(null);
      queryClient.invalidateQueries({ queryKey: SESSIONS_QUERY_KEY });
    },
    onError: () => {
      // Generic message — backend already returns a generic error, but
      // we don't want to surface anything that distinguishes "not found"
      // (another user's id, or stale) from "server error" to the caller.
      toast({ title: t('revokeErrorGeneric'), variant: 'destructive' });
      setPendingRevoke(null);
    },
  });

  const formatWhen = (iso: string) => {
    try {
      return formatDistanceToNow(new Date(iso), { addSuffix: true, locale: dateLocale });
    } catch {
      return iso;
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('title')}</CardTitle>
        <CardDescription>{t('description')}</CardDescription>
      </CardHeader>
      <CardContent>
        {query.isLoading && (
          <p className="text-muted-foreground text-sm" role="status">
            {t('loading')}
          </p>
        )}

        {query.isError && !query.isLoading && (
          <p className="text-destructive text-sm" role="alert">
            {t('loadError')}
          </p>
        )}

        {query.data && query.data.sessions.length === 0 && (
          <p className="text-muted-foreground text-sm">{t('empty')}</p>
        )}

        {query.data && query.data.sessions.length > 0 && (
          <ul className="divide-border divide-y" aria-label={t('title')}>
            {query.data.sessions.map((s) => (
              <li
                key={s.id}
                className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="min-w-0 flex-1 space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="truncate text-sm font-medium">
                      {s.created_user_agent || '—'}
                    </span>
                    {s.current && (
                      <Badge variant="secondary" aria-label={t('currentBadge')}>
                        {t('currentBadge')}
                      </Badge>
                    )}
                  </div>
                  <p className="text-muted-foreground text-xs">
                    {s.created_ip || '—'} · {t('signedInAt', { when: formatWhen(s.created_at) })}
                  </p>
                  <p className="text-muted-foreground text-xs">
                    {t('expiresAt', { when: formatWhen(s.expires_at) })}
                  </p>
                </div>
                {!s.current && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPendingRevoke(s)}
                    disabled={revokeMutation.isPending}
                  >
                    {t('revoke')}
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </CardContent>

      <AlertDialog
        open={pendingRevoke !== null}
        onOpenChange={(open) => {
          if (!open) setPendingRevoke(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('revoke')}</AlertDialogTitle>
            <AlertDialogDescription>{t('revokeConfirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={revokeMutation.isPending}>
              {tCommon('cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => pendingRevoke && revokeMutation.mutate(pendingRevoke.id)}
              disabled={revokeMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t('revoke')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  );
}
