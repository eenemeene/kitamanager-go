'use client';

import { useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, XCircle, Shield, UserCog, Database, Download } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Pagination } from '@/components/ui/pagination';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { QueryError } from '@/components/crud/query-error';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { AuditLogResponse } from '@/lib/api/types';

const PAGE_LIMIT = 20;

// Group the free-form `action` strings into four categories so the table can
// render an icon and tighten filtering.
type Category = 'auth' | 'authz' | 'data' | 'export';

function categoryOf(action: string): Category {
  if (
    action === 'login' ||
    action === 'login_failed' ||
    action === 'logout' ||
    action === 'password_change' ||
    action === 'password_change_failed' ||
    action === 'password_reset'
  ) {
    return 'auth';
  }
  if (
    action === 'role_change' ||
    action === 'user_add_to_org' ||
    action === 'user_remove_from_org' ||
    action === 'superadmin_grant' ||
    action === 'superadmin_revoke'
  ) {
    return 'authz';
  }
  if (action.endsWith('_export')) return 'export';
  return 'data';
}

const CATEGORY_META: Record<
  Category,
  { icon: typeof Shield; className: string; labelKey: string }
> = {
  auth: { icon: Shield, className: 'text-blue-600', labelKey: 'auditLog.categoryAuth' },
  authz: { icon: UserCog, className: 'text-purple-600', labelKey: 'auditLog.categoryAccess' },
  data: { icon: Database, className: 'text-emerald-600', labelKey: 'auditLog.categoryData' },
  export: { icon: Download, className: 'text-amber-600', labelKey: 'auditLog.categoryExport' },
};

function formatTimestamp(iso: string, locale: string): string {
  try {
    return new Date(iso).toLocaleString(locale);
  } catch {
    return iso;
  }
}

function prettifyDetails(details: string | undefined): string {
  if (!details) return '';
  try {
    return JSON.stringify(JSON.parse(details), null, 2);
  } catch {
    return details;
  }
}

export default function AuditLogPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();

  const [page, setPage] = useState(1);
  const [action, setAction] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [selected, setSelected] = useState<AuditLogResponse | null>(null);

  // Keep react-query's cache keyed by every filter so stale results never bleed
  // across changes.
  const queryFilters = useMemo(() => ({ page, action, from, to }), [page, action, from, to]);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: queryKeys.auditLogs.list(orgId, queryFilters),
    queryFn: () =>
      apiClient.getAuditLogs(orgId, {
        page,
        limit: PAGE_LIMIT,
        action: action || undefined,
        from: from || undefined,
        to: to || undefined,
      }),
    enabled: !!orgId,
    // The global 60s staleTime would otherwise serve a stale cache when the
    // user navigates back right after a mutation that produced a new event.
    staleTime: 0,
    refetchOnMount: 'always',
  });

  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_LIMIT)) : 1;

  if (error) {
    return <QueryError error={error} onRetry={() => refetch()} />;
  }

  return (
    <div className="space-y-4 p-3 md:p-6">
      <Card>
        <CardHeader>
          <CardTitle>{t('auditLog.title')}</CardTitle>
          <p className="text-muted-foreground text-sm">{t('auditLog.description')}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-end gap-3">
            <div className="flex flex-col gap-1">
              <Label htmlFor="audit-from" className="text-xs">
                {t('auditLog.from')}
              </Label>
              <Input
                id="audit-from"
                type="date"
                value={from}
                onChange={(e) => {
                  setFrom(e.target.value);
                  setPage(1);
                }}
                className="w-[10rem]"
              />
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="audit-to" className="text-xs">
                {t('auditLog.to')}
              </Label>
              <Input
                id="audit-to"
                type="date"
                value={to}
                onChange={(e) => {
                  setTo(e.target.value);
                  setPage(1);
                }}
                className="w-[10rem]"
              />
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="audit-action" className="text-xs">
                {t('auditLog.action')}
              </Label>
              <Input
                id="audit-action"
                placeholder={t('auditLog.actionPlaceholder')}
                value={action}
                onChange={(e) => {
                  setAction(e.target.value);
                  setPage(1);
                }}
                className="w-[14rem]"
              />
            </div>
            {(from || to || action) && (
              <Button
                variant="ghost"
                onClick={() => {
                  setFrom('');
                  setTo('');
                  setAction('');
                  setPage(1);
                }}
              >
                {t('auditLog.clearFilters')}
              </Button>
            )}
          </div>

          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : data && data.data.length > 0 ? (
            <>
              {/* Desktop table */}
              <div className="hidden md:block">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('auditLog.time')}</TableHead>
                      <TableHead>{t('auditLog.actor')}</TableHead>
                      <TableHead>{t('auditLog.action')}</TableHead>
                      <TableHead className="hidden lg:table-cell">
                        {t('auditLog.resource')}
                      </TableHead>
                      <TableHead className="w-[60px] text-center">{t('auditLog.result')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.data.map((entry) => (
                      <AuditRow key={entry.id} entry={entry} onSelect={() => setSelected(entry)} />
                    ))}
                  </TableBody>
                </Table>
              </div>

              {/* Mobile card list */}
              <ul className="space-y-2 md:hidden">
                {data.data.map((entry) => (
                  <li key={entry.id}>
                    <AuditCard entry={entry} onSelect={() => setSelected(entry)} />
                  </li>
                ))}
              </ul>

              <Pagination
                page={page}
                totalPages={totalPages}
                total={data.total}
                limit={PAGE_LIMIT}
                onPageChange={setPage}
              />
            </>
          ) : (
            <p className="text-muted-foreground py-12 text-center text-sm">{t('auditLog.empty')}</p>
          )}
        </CardContent>
      </Card>

      <AuditDetailDialog entry={selected} onClose={() => setSelected(null)} />
    </div>
  );
}

interface AuditRowProps {
  entry: AuditLogResponse;
  onSelect: () => void;
}

function AuditRow({ entry, onSelect }: AuditRowProps) {
  const t = useTranslations();
  const locale = useLocale() === 'de' ? 'de-DE' : 'en-US';
  const category = categoryOf(entry.action);
  const meta = CATEGORY_META[category];
  const Icon = meta.icon;

  return (
    <TableRow
      className="hover:bg-muted cursor-pointer"
      onClick={onSelect}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onSelect();
        }
      }}
    >
      <TableCell className="text-sm whitespace-nowrap">
        {formatTimestamp(entry.timestamp, locale)}
      </TableCell>
      <TableCell className="text-sm">{entry.user_email || '—'}</TableCell>
      <TableCell>
        <span className="inline-flex items-center gap-2 text-sm">
          <Icon className={`h-4 w-4 shrink-0 ${meta.className}`} aria-label={t(meta.labelKey)} />
          <span className="font-medium">{entry.action}</span>
        </span>
      </TableCell>
      <TableCell className="text-muted-foreground hidden text-sm lg:table-cell">
        {entry.resource_type
          ? `${entry.resource_type}${entry.resource_id ? ` #${entry.resource_id}` : ''}`
          : '—'}
      </TableCell>
      <TableCell className="text-center">
        {entry.success ? (
          <CheckCircle2 className="inline h-4 w-4 text-emerald-600" />
        ) : (
          <XCircle className="inline h-4 w-4 text-red-600" />
        )}
      </TableCell>
    </TableRow>
  );
}

function AuditCard({ entry, onSelect }: AuditRowProps) {
  const t = useTranslations();
  const locale = useLocale() === 'de' ? 'de-DE' : 'en-US';
  const category = categoryOf(entry.action);
  const meta = CATEGORY_META[category];
  const Icon = meta.icon;

  return (
    <button
      type="button"
      onClick={onSelect}
      className="hover:bg-muted flex w-full flex-col gap-1 rounded border px-3 py-2 text-left"
    >
      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <span>{formatTimestamp(entry.timestamp, locale)}</span>
        {entry.success ? (
          <CheckCircle2 className="h-4 w-4 text-emerald-600" />
        ) : (
          <XCircle className="h-4 w-4 text-red-600" />
        )}
      </div>
      <div className="flex items-center gap-2 text-sm font-medium">
        <Icon className={`h-4 w-4 ${meta.className}`} aria-label={t(meta.labelKey)} />
        <span>{entry.action}</span>
      </div>
      <div className="text-muted-foreground text-xs">
        {entry.user_email || '—'}
        {entry.resource_type &&
          ` · ${entry.resource_type}${entry.resource_id ? ` #${entry.resource_id}` : ''}`}
      </div>
    </button>
  );
}

interface AuditDetailDialogProps {
  entry: AuditLogResponse | null;
  onClose: () => void;
}

function AuditDetailDialog({ entry, onClose }: AuditDetailDialogProps) {
  const t = useTranslations();
  const locale = useLocale() === 'de' ? 'de-DE' : 'en-US';

  return (
    <Dialog open={!!entry} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('auditLog.detailTitle')}</DialogTitle>
          <DialogDescription className="sr-only">{t('auditLog.detailTitle')}</DialogDescription>
        </DialogHeader>
        {entry && (
          <dl className="space-y-2 text-sm">
            <DetailRow
              label={t('auditLog.time')}
              value={formatTimestamp(entry.timestamp, locale)}
            />
            <DetailRow
              label={t('auditLog.result')}
              value={entry.success ? t('auditLog.success') : t('auditLog.failed')}
            />
            <DetailRow label={t('auditLog.action')} value={entry.action} />
            <DetailRow label={t('auditLog.actor')} value={entry.user_email || '—'} />
            {entry.resource_type && (
              <DetailRow
                label={t('auditLog.resource')}
                value={`${entry.resource_type}${entry.resource_id ? ` #${entry.resource_id}` : ''}`}
              />
            )}
            {entry.ip_address && (
              <DetailRow label={t('auditLog.ipAddress')} value={entry.ip_address} />
            )}
            {entry.details && (
              <div className="pt-2">
                <dt className="text-muted-foreground mb-1 text-xs font-semibold uppercase">
                  {t('auditLog.details')}
                </dt>
                <dd>
                  <pre className="bg-muted max-h-64 overflow-auto rounded p-2 text-xs leading-snug">
                    {prettifyDetails(entry.details)}
                  </pre>
                </dd>
              </div>
            )}
          </dl>
        )}
      </DialogContent>
    </Dialog>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4">
      <dt className="text-muted-foreground text-xs font-semibold uppercase">{label}</dt>
      <dd className="text-right text-sm">{value}</dd>
    </div>
  );
}
