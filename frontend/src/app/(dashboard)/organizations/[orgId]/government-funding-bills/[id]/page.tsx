'use client';

import { Fragment, useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  MinusCircle,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { formatCurrency } from '@/lib/utils/formatting';
import type { FundingComparisonChild } from '@/lib/api/types';

function StatusBadge({
  status,
  t,
}: {
  status: FundingComparisonChild['status'];
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
    case 'bill_only':
      return (
        <Badge variant="warning">
          <AlertTriangle className="mr-1 h-3 w-3" />
          {t('statusBillOnly')}
        </Badge>
      );
    case 'calc_only':
      return (
        <Badge variant="secondary">
          <MinusCircle className="mr-1 h-3 w-3" />
          {t('statusCalcOnly')}
        </Badge>
      );
  }
}

export default function GovernmentFundingBillDetailPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const id = Number(params.id);
  const t = useTranslations('governmentFundingBills');
  const tCommon = useTranslations('common');
  const [expandedChild, setExpandedChild] = useState<string | null>(null);

  const { data: result, isLoading } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.detail(orgId, id),
    queryFn: () => apiClient.getGovernmentFundingBillPeriod(orgId, id),
  });

  const {
    data: comparison,
    isLoading: comparisonLoading,
    isError: comparisonError,
  } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.compare(orgId, id),
    queryFn: () => apiClient.compareGovernmentFundingBill(orgId, id),
    retry: false,
  });

  if (isLoading) {
    return <p className="text-muted-foreground py-8 text-center">{tCommon('loading')}</p>;
  }

  if (!result) {
    return <p className="text-muted-foreground py-8 text-center">{tCommon('notFound')}</p>;
  }

  // Build a map of comparison children by voucher number for quick lookup
  const comparisonByVoucher = new Map<string, FundingComparisonChild>();
  if (comparison?.children) {
    for (const child of comparison.children) {
      comparisonByVoucher.set(child.voucher_number, child);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/organizations/${orgId}/government-funding-bills`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">{result.facility_name}</h1>
          <p className="text-muted-foreground text-sm">
            {new Date(result.from).toLocaleDateString('de-DE', {
              month: 'long',
              year: 'numeric',
            })}
            {' \u2014 '}
            {result.file_name}
          </p>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-muted-foreground text-sm font-medium">
              {t('facilityTotal')}
              {comparison && ' / ' + t('calculatedTotal')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-lg font-semibold">
              {formatCurrency(result.facility_total)}
              {comparison && (
                <>
                  {' / '}
                  {formatCurrency(comparison.calculated_total)}
                </>
              )}
            </p>
            {comparison && (
              <p
                className={`text-sm ${
                  comparison.difference_count === 0 &&
                  comparison.bill_only_count === 0 &&
                  comparison.calc_only_count === 0
                    ? 'text-green-600'
                    : 'text-red-600'
                }`}
              >
                {t('difference')}: {formatCurrency(comparison.difference)}
              </p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-muted-foreground text-sm font-medium">
              {t('contractBooking')} / {t('correctionBooking')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-lg font-semibold">
              {formatCurrency(result.contract_booking)} /{' '}
              {formatCurrency(result.correction_booking)}
            </p>
          </CardContent>
        </Card>
        {comparison ? (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-muted-foreground text-sm font-medium">
                {t('comparison')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                <Badge
                  variant={
                    comparison.difference_count === 0 &&
                    comparison.bill_only_count === 0 &&
                    comparison.calc_only_count === 0
                      ? 'success'
                      : 'destructive'
                  }
                >
                  {t('matchCount')}: {comparison.match_count}
                </Badge>
                {comparison.difference_count > 0 && (
                  <Badge variant="destructive">
                    {t('differenceCount')}: {comparison.difference_count}
                  </Badge>
                )}
                {comparison.bill_only_count > 0 && (
                  <Badge variant="warning">
                    {t('billOnlyCount')}: {comparison.bill_only_count}
                  </Badge>
                )}
                {comparison.calc_only_count > 0 && (
                  <Badge variant="secondary">
                    {t('calcOnlyCount')}: {comparison.calc_only_count}
                  </Badge>
                )}
              </div>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-muted-foreground text-sm font-medium">
                {t('matchedChildren')} / {t('unmatchedChildren')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg font-semibold">
                <span className="text-green-600">{result.matched_count}</span>
                {' / '}
                <span className="text-red-600">{result.unmatched_count}</span>
                <span className="text-muted-foreground ml-2 text-sm">
                  ({result.children_count} {t('children')})
                </span>
              </p>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Comparison info banner */}
      {comparisonLoading && (
        <p className="text-muted-foreground text-center text-sm">{t('comparisonLoading')}</p>
      )}
      {comparisonError && (
        <div className="rounded-md border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800 dark:border-yellow-800 dark:bg-yellow-950 dark:text-yellow-200">
          <AlertTriangle className="mr-2 inline h-4 w-4" />
          {t('comparisonError')}
        </div>
      )}

      {/* Surcharges */}
      {result.surcharges && result.surcharges.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>{t('surcharges')}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 md:grid-cols-3">
              {result.surcharges.map((s) => (
                <div
                  key={`${s.key}-${s.value}`}
                  className="flex justify-between rounded-md border p-3"
                >
                  <span className="text-muted-foreground text-sm">
                    {s.key}: {s.value}
                  </span>
                  <span className="font-medium">{formatCurrency(s.amount)}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Children Table */}
      <Card>
        <CardHeader>
          <CardTitle>
            {t('children')} ({result.children_count})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('voucherNumber')}</TableHead>
                <TableHead>{t('childName')}</TableHead>
                <TableHead className="hidden md:table-cell">{t('birthDate')}</TableHead>
                <TableHead className="text-right">{t('billAmount')}</TableHead>
                {comparison && (
                  <>
                    <TableHead className="hidden text-right md:table-cell">
                      {t('calculatedAmount')}
                    </TableHead>
                    <TableHead className="hidden text-right md:table-cell">
                      {t('difference')}
                    </TableHead>
                    <TableHead>{t('comparisonStatus')}</TableHead>
                  </>
                )}
                {!comparison && <TableHead>{t('matched')}</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {result.children.map((child) => {
                const comp = comparisonByVoucher.get(child.voucher_number);
                const isExpanded = expandedChild === child.voucher_number;

                return (
                  <Fragment key={child.voucher_number}>
                    <TableRow
                      className={comp?.properties?.length ? 'cursor-pointer' : ''}
                      onClick={() => {
                        if (comp?.properties?.length) {
                          setExpandedChild(isExpanded ? null : child.voucher_number);
                        }
                      }}
                    >
                      <TableCell className="font-mono text-sm">
                        {comp?.properties?.length ? (
                          <span className="flex items-center gap-1">
                            {isExpanded ? (
                              <ChevronDown className="h-3 w-3" />
                            ) : (
                              <ChevronRight className="h-3 w-3" />
                            )}
                            {child.voucher_number}
                          </span>
                        ) : (
                          child.voucher_number
                        )}
                      </TableCell>
                      <TableCell>
                        {(child.matched || comp?.child_id) && child.child_id ? (
                          <Link
                            href={`/organizations/${orgId}/children/${child.child_id}`}
                            className="text-primary hover:underline"
                            onClick={(e) => e.stopPropagation()}
                          >
                            {child.child_name}
                          </Link>
                        ) : (
                          child.child_name
                        )}
                      </TableCell>
                      <TableCell className="hidden md:table-cell">{child.birth_date}</TableCell>
                      <TableCell className="text-right">
                        {formatCurrency(child.total_amount)}
                      </TableCell>
                      {comparison && comp && (
                        <>
                          <TableCell className="hidden text-right md:table-cell">
                            {comp.calculated_total != null
                              ? formatCurrency(comp.calculated_total)
                              : '\u2014'}
                          </TableCell>
                          <TableCell className="hidden text-right md:table-cell">
                            <span
                              className={comp.difference === 0 ? 'text-green-600' : 'text-red-600'}
                            >
                              {comp.difference != null ? formatCurrency(comp.difference) : '\u2014'}
                            </span>
                          </TableCell>
                          <TableCell>
                            <StatusBadge status={comp.status} t={t} />
                          </TableCell>
                        </>
                      )}
                      {comparison && !comp && (
                        <>
                          <TableCell className="text-muted-foreground hidden text-right md:table-cell">
                            &mdash;
                          </TableCell>
                          <TableCell className="text-muted-foreground hidden text-right md:table-cell">
                            &mdash;
                          </TableCell>
                          <TableCell>
                            {child.matched ? (
                              <Badge variant="success">
                                <CheckCircle2 className="mr-1 h-3 w-3" />
                              </Badge>
                            ) : (
                              <Badge variant="destructive">
                                <XCircle className="mr-1 h-3 w-3" />
                              </Badge>
                            )}
                          </TableCell>
                        </>
                      )}
                      {!comparison && (
                        <TableCell>
                          {child.matched ? (
                            <Badge variant="success">
                              <CheckCircle2 className="mr-1 h-3 w-3" />
                            </Badge>
                          ) : (
                            <Badge variant="destructive">
                              <XCircle className="mr-1 h-3 w-3" />
                            </Badge>
                          )}
                        </TableCell>
                      )}
                    </TableRow>
                    {/* Expandable property detail */}
                    {isExpanded && comp?.properties && (
                      <TableRow key={`${child.voucher_number}-properties`}>
                        <TableCell colSpan={comparison ? 7 : 5} className="bg-muted/50 p-0">
                          <div className="p-3 md:pl-10">
                            <Table>
                              <TableHeader>
                                <TableRow>
                                  <TableHead className="text-xs">{t('surcharges')}</TableHead>
                                  <TableHead className="text-right text-xs">
                                    {t('billAmount')}
                                  </TableHead>
                                  <TableHead className="text-right text-xs">
                                    {t('calculatedAmount')}
                                  </TableHead>
                                  <TableHead className="text-right text-xs">
                                    {t('difference')}
                                  </TableHead>
                                </TableRow>
                              </TableHeader>
                              <TableBody>
                                {comp.properties.map((prop) => (
                                  <TableRow key={`${prop.key}-${prop.value}`}>
                                    <TableCell className="text-sm">
                                      {prop.label || `${prop.key}: ${prop.value}`}
                                    </TableCell>
                                    <TableCell className="text-right text-sm">
                                      {prop.bill_amount != null
                                        ? formatCurrency(prop.bill_amount)
                                        : '\u2014'}
                                    </TableCell>
                                    <TableCell className="text-right text-sm">
                                      {prop.calculated_amount != null
                                        ? formatCurrency(prop.calculated_amount)
                                        : '\u2014'}
                                    </TableCell>
                                    <TableCell className="text-right text-sm">
                                      <span
                                        className={
                                          prop.difference === 0 ? 'text-green-600' : 'text-red-600'
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
                  </Fragment>
                );
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
