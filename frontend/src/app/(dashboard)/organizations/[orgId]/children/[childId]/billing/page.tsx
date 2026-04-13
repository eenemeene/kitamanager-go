'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import {
  CheckCircle2,
  XCircle,
  AlertTriangle,
  MinusCircle,
  FileWarning,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { Breadcrumb } from '@/components/ui/breadcrumb';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { QueryError } from '@/components/crud/query-error';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { ChildBillingHistoryEntry } from '@/lib/api/types';
import { formatCurrency, formatDate } from '@/lib/utils/formatting';

function StatusBadge({
  status,
  t,
}: {
  status: ChildBillingHistoryEntry['status'];
  t: (key: string) => string;
}) {
  switch (status) {
    case 'match':
      return (
        <Badge variant="success">
          <CheckCircle2 className="mr-1 h-3 w-3" />
          {t('statusMatch')}
        </Badge>
      );
    case 'difference':
      return (
        <Badge variant="destructive">
          <XCircle className="mr-1 h-3 w-3" />
          {t('statusDifference')}
        </Badge>
      );
    case 'no_contract':
      return (
        <Badge variant="warning">
          <AlertTriangle className="mr-1 h-3 w-3" />
          {t('statusNoContract')}
        </Badge>
      );
    case 'no_funding_config':
      return (
        <Badge variant="secondary">
          <FileWarning className="mr-1 h-3 w-3" />
          {t('statusNoFundingConfig')}
        </Badge>
      );
    case 'bill_only':
      return (
        <Badge variant="warning">
          <MinusCircle className="mr-1 h-3 w-3" />
          {t('statusBillOnly')}
        </Badge>
      );
  }
}

function BillingRow({
  entry,
  t,
  tCommon,
  tLabels,
}: {
  entry: ChildBillingHistoryEntry;
  t: (key: string) => string;
  tCommon: (key: string) => string;
  tLabels: { has: (key: string) => boolean; (key: string): string };
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <>
      <TableRow className="hover:bg-muted/50 cursor-pointer" onClick={() => setExpanded(!expanded)}>
        <TableCell>
          {expanded ? (
            <ChevronDown className="mr-1 inline h-4 w-4" />
          ) : (
            <ChevronRight className="mr-1 inline h-4 w-4" />
          )}
          {formatDate(entry.bill_from)}
        </TableCell>
        <TableCell className="hidden md:table-cell">{entry.facility_name}</TableCell>
        <TableCell className="hidden font-mono text-xs lg:table-cell">
          {entry.voucher_number}
        </TableCell>
        <TableCell className="hidden md:table-cell">
          {entry.age != null ? entry.age : '\u2014'}
        </TableCell>
        <TableCell className="text-right">{formatCurrency(entry.bill_total)}</TableCell>
        <TableCell className="hidden text-right md:table-cell">
          {entry.correction_total ? (
            <span className="text-blue-600">{formatCurrency(entry.correction_total)}</span>
          ) : (
            '\u2014'
          )}
        </TableCell>
        <TableCell className="hidden text-right md:table-cell">
          {entry.calculated_total != null ? formatCurrency(entry.calculated_total) : '\u2014'}
        </TableCell>
        <TableCell className="hidden text-right md:table-cell">
          {entry.difference != null ? (
            <span
              className={
                entry.difference < 0 ? 'text-red-600' : entry.difference > 0 ? 'text-green-600' : ''
              }
            >
              {formatCurrency(entry.difference)}
            </span>
          ) : (
            '\u2014'
          )}
        </TableCell>
        <TableCell className="hidden text-right md:table-cell">
          <span
            className={
              entry.running_difference < 0
                ? 'font-medium text-red-600'
                : entry.running_difference > 0
                  ? 'font-medium text-green-600'
                  : ''
            }
          >
            {formatCurrency(entry.running_difference)}
          </span>
        </TableCell>
        <TableCell>
          <StatusBadge status={entry.status} t={t} />
        </TableCell>
      </TableRow>
      {expanded && (
        <TableRow>
          <TableCell colSpan={10} className="p-0">
            <div className="bg-muted/30 p-3 md:p-4">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{tCommon('name')}</TableHead>
                    <TableHead className="text-right">{t('billAmount')}</TableHead>
                    <TableHead className="text-right">{t('calculatedAmount')}</TableHead>
                    <TableHead className="text-right">{t('difference')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {entry.properties.map((prop, pIdx) => (
                    <TableRow key={pIdx}>
                      <TableCell>
                        {prop.label ||
                          (tLabels.has(`${prop.key}--${prop.value}`)
                            ? tLabels(`${prop.key}--${prop.value}`)
                            : `${prop.key}: ${prop.value}`)}
                      </TableCell>
                      <TableCell className="text-right">
                        {prop.bill_amount != null ? formatCurrency(prop.bill_amount) : '\u2014'}
                      </TableCell>
                      <TableCell className="text-right">
                        {prop.calculated_amount != null
                          ? formatCurrency(prop.calculated_amount)
                          : '\u2014'}
                      </TableCell>
                      <TableCell className="text-right">
                        <span
                          className={
                            prop.difference < 0
                              ? 'text-red-600'
                              : prop.difference > 0
                                ? 'text-green-600'
                                : ''
                          }
                        >
                          {formatCurrency(prop.difference)}
                        </span>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

export default function ChildBillingHistoryPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const childId = Number(params.childId);
  const t = useTranslations('governmentFundingBills');
  const tChildren = useTranslations('children');
  const tCommon = useTranslations('common');
  const tNav = useTranslations('nav');
  const tLabels = useTranslations('fundingLabels');

  const {
    data: history,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: queryKeys.children.billingHistory(orgId, childId),
    queryFn: () => apiClient.getChildBillingHistory(orgId, childId),
    enabled: !!orgId && !!childId,
  });

  const breadcrumbs = [
    { label: tNav('children'), href: `/organizations/${orgId}/children` },
    { label: history?.child_name ?? '...' },
    { label: tChildren('billingHistory') },
  ];

  if (error) {
    return (
      <div className="space-y-4">
        <Breadcrumb items={breadcrumbs} />
        <QueryError error={error} onRetry={refetch} />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Breadcrumb items={breadcrumbs} />

      {isLoading ? (
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-48" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-64 w-full" />
          </CardContent>
        </Card>
      ) : history ? (
        <>
          {/* Summary card */}
          <Card>
            <CardHeader>
              <CardTitle>{tChildren('billingHistory')}</CardTitle>
              <CardDescription>{history.child_name}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                <div>
                  <p className="text-muted-foreground text-sm">{t('voucherNumbers')}</p>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {history.voucher_numbers.length > 0 ? (
                      history.voucher_numbers.map((v) => (
                        <Badge key={v} variant="outline">
                          {v}
                        </Badge>
                      ))
                    ) : (
                      <span className="text-muted-foreground text-sm">
                        {tChildren('noVoucherNumbers')}
                      </span>
                    )}
                  </div>
                </div>
                <div>
                  <p className="text-muted-foreground text-sm">{t('billedTotal')}</p>
                  <p className="text-lg font-semibold">{formatCurrency(history.total_billed)}</p>
                </div>
                <div>
                  <p className="text-muted-foreground text-sm">{t('difference')}</p>
                  <p
                    className={`text-lg font-semibold ${history.total_difference < 0 ? 'text-red-600' : history.total_difference > 0 ? 'text-green-600' : ''}`}
                  >
                    {formatCurrency(history.total_difference)}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Billing entries table */}
          <Card>
            <CardContent className="p-0">
              {history.entries.length === 0 ? (
                <div className="text-muted-foreground p-6 text-center">
                  {tChildren('noBillingEntries')}
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('billingMonth')}</TableHead>
                      <TableHead className="hidden md:table-cell">{t('facilityName')}</TableHead>
                      <TableHead className="hidden lg:table-cell">{t('voucherNumber')}</TableHead>
                      <TableHead className="hidden md:table-cell">{t('age')}</TableHead>
                      <TableHead className="text-right">{t('billTotal')}</TableHead>
                      <TableHead className="hidden text-right md:table-cell">
                        {t('correctionTotal')}
                      </TableHead>
                      <TableHead className="hidden text-right md:table-cell">
                        {t('calcTotal')}
                      </TableHead>
                      <TableHead className="hidden text-right md:table-cell">
                        {t('difference')}
                      </TableHead>
                      <TableHead className="hidden text-right md:table-cell">
                        {t('runningDifference')}
                      </TableHead>
                      <TableHead>{tCommon('status')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {history.entries.map((entry, idx) => (
                      <BillingRow
                        key={idx}
                        entry={entry}
                        t={t}
                        tCommon={tCommon}
                        tLabels={tLabels}
                      />
                    ))}
                  </TableBody>

                  {/* Summary footer */}
                  {history.entries.length > 1 && (
                    <tfoot>
                      <TableRow className="font-semibold">
                        <TableCell>{tCommon('total')}</TableCell>
                        <TableCell className="hidden md:table-cell" />
                        <TableCell className="hidden lg:table-cell" />
                        <TableCell className="hidden md:table-cell" />
                        <TableCell className="text-right">
                          {formatCurrency(history.total_billed)}
                        </TableCell>
                        <TableCell className="hidden md:table-cell" />
                        <TableCell className="hidden text-right md:table-cell">
                          {formatCurrency(history.total_calculated)}
                        </TableCell>
                        <TableCell className="hidden text-right md:table-cell">
                          <span
                            className={
                              history.total_difference < 0
                                ? 'text-red-600'
                                : history.total_difference > 0
                                  ? 'text-green-600'
                                  : ''
                            }
                          >
                            {formatCurrency(history.total_difference)}
                          </span>
                        </TableCell>
                        <TableCell className="hidden md:table-cell" />
                        <TableCell />
                      </TableRow>
                    </tfoot>
                  )}
                </Table>
              )}
            </CardContent>
          </Card>
        </>
      ) : null}
    </div>
  );
}
