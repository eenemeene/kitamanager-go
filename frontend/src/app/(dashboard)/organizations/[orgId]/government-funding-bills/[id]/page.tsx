'use client';

import { Fragment, useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import {
  CheckCircle2,
  XCircle,
  AlertTriangle,
  MinusCircle,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Breadcrumb } from '@/components/ui/breadcrumb';
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

import type { FundingComparisonChild, MismatchType } from '@/lib/api/types';
import { useFormatters } from '@/hooks/use-formatters';

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

function MismatchTag({ mismatch, t }: { mismatch: MismatchType; t: (key: string) => string }) {
  switch (mismatch) {
    case 'missing':
      return (
        <span className="bg-warning/15 text-warning inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium">
          {t('mismatchMissing')}
        </span>
      );
    case 'additional':
      return (
        <span className="bg-info/15 text-info inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium">
          {t('mismatchAdditional')}
        </span>
      );
    case 'different':
      return (
        <span className="bg-destructive/15 text-destructive inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium">
          {t('mismatchDifferent')}
        </span>
      );
  }
}

/** Summarize the mismatch reasons for a child's properties into a compact description. */
function DifferenceReason({
  comp,
  t,
}: {
  comp: FundingComparisonChild;
  t: (key: string, values?: Record<string, number>) => string;
}) {
  if (comp.status === 'bill_only') {
    return <span className="text-muted-foreground text-xs">{t('mismatchReasonBillOnly')}</span>;
  }
  if (comp.status === 'calc_only') {
    return <span className="text-muted-foreground text-xs">{t('mismatchReasonCalcOnly')}</span>;
  }
  if (comp.status !== 'difference' || !comp.properties?.length) return null;

  const mismatchProps = comp.properties.filter((p) => p.mismatch);
  const rateProps = comp.properties.filter((p) => !p.mismatch && p.difference !== 0);

  const reasons: string[] = [];
  if (mismatchProps.length > 0) {
    reasons.push(t('mismatchCount', { count: mismatchProps.length }));
  }
  if (rateProps.length > 0) {
    reasons.push(t('mismatchReasonRate'));
  }

  if (reasons.length === 0) return null;

  return <span className="text-muted-foreground text-xs">{reasons.join(' + ')}</span>;
}

export default function GovernmentFundingBillDetailPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const id = Number(params.id);
  const t = useTranslations('governmentFundingBills');
  const fmt = useFormatters();
  const tCommon = useTranslations('common');
  const tLabels = useTranslations('fundingLabels');
  const [expandedChild, setExpandedChild] = useState<string | null>(null);

  const { data: result, isLoading } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.detail(orgId, id),
    queryFn: () => apiClient.getGovernmentFundingBillPeriod(orgId, id),
  });

  const translateLabel = (key: string, value: string, fallbackLabel?: string) => {
    const translationKey = `${key}--${value}`;
    const translated = tLabels.has(translationKey) ? tLabels(translationKey) : null;
    return translated || fallbackLabel || value;
  };

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
      comparisonByVoucher.set(child.voucher_number ?? '', child);
    }
  }

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <Breadcrumb
          items={[
            { label: t('title'), href: `/organizations/${orgId}/government-funding-bills` },
            { label: result.facility_name },
          ]}
        />
        <h1 className="text-2xl font-bold">{result.facility_name}</h1>
        <p className="text-muted-foreground text-sm">
          {fmt.monthYear(result.from, { month: 'long', year: 'numeric' })}
          {' \u2014 '}
          {result.file_name}
        </p>
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
              {fmt.currency(result.facility_total)}
              {comparison && (
                <>
                  {' / '}
                  {fmt.currency(comparison.calculated_total)}
                </>
              )}
            </p>
            {comparison && (
              <p
                className={`text-sm ${
                  comparison.difference_count === 0 &&
                  comparison.bill_only_count === 0 &&
                  comparison.calc_only_count === 0
                    ? 'text-success'
                    : 'text-destructive'
                }`}
              >
                {t('difference')}: {fmt.currency(comparison.difference)}
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
              {fmt.currency(result.contract_booking)} / {fmt.currency(result.correction_booking)}
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
              <ul className="space-y-1 text-sm">
                <li className="flex items-center gap-2">
                  <span
                    className={`inline-block h-2.5 w-2.5 rounded-full ${
                      comparison.difference_count === 0 &&
                      comparison.bill_only_count === 0 &&
                      comparison.calc_only_count === 0
                        ? 'bg-success'
                        : 'bg-muted-foreground'
                    }`}
                  />
                  <span className="text-muted-foreground">{t('matchCount')}</span>
                  <span className="font-medium">
                    {t('childCount', { count: comparison.match_count })}
                  </span>
                </li>
                {comparison.difference_count > 0 && (
                  <li className="flex items-center gap-2">
                    <span className="bg-destructive inline-block h-2.5 w-2.5 rounded-full" />
                    <span className="text-muted-foreground">{t('differenceCount')}</span>
                    <span className="font-medium">
                      {t('childCount', { count: comparison.difference_count })}
                      {' · '}
                      {fmt.currency(
                        comparison.children
                          .filter((c) => c.status === 'difference')
                          .reduce((sum, c) => sum + (c.bill_total ?? 0), 0)
                      )}
                    </span>
                  </li>
                )}
                {comparison.bill_only_count > 0 && (
                  <li className="flex items-center gap-2">
                    <span className="bg-warning inline-block h-2.5 w-2.5 rounded-full" />
                    <span className="text-muted-foreground">{t('billOnlyCount')}</span>
                    <span className="font-medium">
                      {t('childCount', { count: comparison.bill_only_count })}
                    </span>
                  </li>
                )}
                {comparison.calc_only_count > 0 && (
                  <li className="flex items-center gap-2">
                    <span className="bg-destructive inline-block h-2.5 w-2.5 rounded-full" />
                    <span className="text-muted-foreground">{t('calcOnlyCount')}</span>
                    <span className="font-medium">
                      {t('childCount', { count: comparison.calc_only_count })}
                      {' · '}
                      {fmt.currency(
                        comparison.children
                          .filter((c) => c.status === 'calc_only')
                          .reduce((sum, c) => sum + (c.calculated_total || 0), 0)
                      )}
                    </span>
                  </li>
                )}
              </ul>
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
                <span className="text-success">{result.matched_count}</span>
                {' / '}
                <span className="text-destructive">{result.unmatched_count}</span>
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
        <div className="border-warning/30 bg-warning/10 text-warning rounded-md border p-3 text-sm">
          <AlertTriangle className="mr-2 inline h-4 w-4" />
          {t('comparisonError')}
        </div>
      )}

      {/* Children Table */}
      <Card>
        <CardHeader>
          <CardTitle>
            {t('childrenInBill')} ({result.children_count})
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
                      {t('correctionTotal')}
                    </TableHead>
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
              {result.children.map((child, idx) => {
                const comp = comparisonByVoucher.get(child.voucher_number);
                const isExpanded = expandedChild === child.voucher_number;
                const hasMultipleRows = child.rows && child.rows.length > 1;
                const isExpandable = hasMultipleRows || (comp?.properties?.length ?? 0) > 0;

                return (
                  <Fragment key={`${child.voucher_number}-${idx}`}>
                    <TableRow
                      className={isExpandable ? 'cursor-pointer' : ''}
                      onClick={() => {
                        if (isExpandable) {
                          setExpandedChild(isExpanded ? null : child.voucher_number);
                        }
                      }}
                    >
                      <TableCell className="font-mono text-sm">
                        {isExpandable ? (
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
                            href={`/organizations/${orgId}/children/${child.child_id}/billing`}
                            className="hover:text-primary hover:underline"
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
                        {fmt.currency(child.total_amount)}
                      </TableCell>
                      {comparison && comp && (
                        <>
                          <TableCell className="hidden text-right md:table-cell">
                            {comp.correction_total ? (
                              <span className="text-info">
                                {fmt.currency(comp.correction_total)}
                              </span>
                            ) : (
                              '\u2014'
                            )}
                          </TableCell>
                          <TableCell className="hidden text-right md:table-cell">
                            {comp.calculated_total != null
                              ? fmt.currency(comp.calculated_total)
                              : '\u2014'}
                          </TableCell>
                          <TableCell className="hidden text-right md:table-cell">
                            <span
                              className={
                                comp.difference != null && comp.difference >= 0
                                  ? 'text-success'
                                  : 'text-destructive'
                              }
                            >
                              {comp.difference != null ? fmt.currency(comp.difference) : '\u2014'}
                            </span>
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-col gap-1">
                              <StatusBadge status={comp.status} t={t} />
                              <DifferenceReason comp={comp} t={t} />
                            </div>
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
                    {/* Expandable detail: rows + comparison properties */}
                    {isExpanded && (
                      <TableRow key={`${child.voucher_number}-${idx}-detail`}>
                        <TableCell colSpan={comparison ? 7 : 5} className="bg-muted/50 p-0">
                          <div className="p-3 md:pl-10">
                            {/* Row-grouped amounts */}
                            {hasMultipleRows &&
                              (() => {
                                const labelMap = new Map<string, string>();
                                if (comp?.properties) {
                                  for (const prop of comp.properties) {
                                    if (prop.label)
                                      labelMap.set(`${prop.key}:${prop.value}`, prop.label);
                                  }
                                }
                                return (
                                  <div>
                                    {child.rows.map((row, rowIdx) => (
                                      <div
                                        key={rowIdx}
                                        className={rowIdx > 0 ? 'mt-2 border-t pt-2' : ''}
                                      >
                                        <div className="flex justify-end py-1">
                                          <span className="text-sm font-bold">
                                            {fmt.currency(row.total_row_amount)}
                                          </span>
                                        </div>
                                        {row.amounts.map((amt, amtIdx) => (
                                          <div
                                            key={amtIdx}
                                            className="text-muted-foreground flex justify-between py-0.5 text-sm"
                                          >
                                            <span>
                                              {translateLabel(
                                                amt.key ?? '',
                                                amt.value ?? '',
                                                labelMap.get(`${amt.key}:${amt.value}`)
                                              )}
                                            </span>
                                            <span>{fmt.currency(amt.amount)}</span>
                                          </div>
                                        ))}
                                      </div>
                                    ))}
                                  </div>
                                );
                              })()}
                            {/* Comparison properties table */}
                            {comp?.properties && comp.properties.length > 0 && (
                              <div className={hasMultipleRows ? 'mt-3 border-t pt-3' : ''}>
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
                                          <span className="flex items-center gap-2">
                                            {translateLabel(
                                              prop.key ?? '',
                                              prop.value ?? '',
                                              prop.label
                                            )}
                                            {prop.mismatch && (
                                              <MismatchTag mismatch={prop.mismatch} t={t} />
                                            )}
                                          </span>
                                        </TableCell>
                                        <TableCell className="text-right text-sm">
                                          {prop.bill_amount != null
                                            ? fmt.currency(prop.bill_amount)
                                            : '\u2014'}
                                        </TableCell>
                                        <TableCell className="text-right text-sm">
                                          {prop.calculated_amount != null
                                            ? fmt.currency(prop.calculated_amount)
                                            : '\u2014'}
                                        </TableCell>
                                        <TableCell className="text-right text-sm">
                                          <span
                                            className={
                                              (prop.difference ?? 0) >= 0
                                                ? 'text-success'
                                                : 'text-destructive'
                                            }
                                          >
                                            {fmt.currency(prop.difference ?? 0)}
                                          </span>
                                        </TableCell>
                                      </TableRow>
                                    ))}
                                  </TableBody>
                                </Table>
                              </div>
                            )}
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

      {/* System-Only Children Table */}
      {comparison &&
        comparison.calc_only_count > 0 &&
        (() => {
          const calcOnlyChildren = comparison.children.filter((c) => c.status === 'calc_only');
          return (
            <Card>
              <CardHeader>
                <CardTitle>
                  {t('systemOnlyChildren')} ({calcOnlyChildren.length})
                </CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('childName')}</TableHead>
                      <TableHead className="hidden md:table-cell">{t('voucherNumber')}</TableHead>
                      <TableHead className="hidden md:table-cell">{t('age')}</TableHead>
                      <TableHead className="text-right">{t('calculatedAmount')}</TableHead>
                      <TableHead className="hidden md:table-cell">{t('contractPeriod')}</TableHead>
                      <TableHead>{t('billHistory')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {calcOnlyChildren.map((child, idx) => (
                      <TableRow key={`${child.voucher_number || child.child_name}-${idx}`}>
                        <TableCell>
                          {child.child_id ? (
                            <Link
                              href={`/organizations/${orgId}/children/${child.child_id}/billing`}
                              className="hover:text-primary hover:underline"
                            >
                              {child.child_name}
                            </Link>
                          ) : (
                            child.child_name
                          )}
                        </TableCell>
                        <TableCell className="hidden font-mono text-sm md:table-cell">
                          {child.voucher_number || '\u2014'}
                        </TableCell>
                        <TableCell className="hidden md:table-cell">
                          {child.age != null ? child.age : '\u2014'}
                        </TableCell>
                        <TableCell className="text-right">
                          {child.calculated_total != null
                            ? fmt.currency(child.calculated_total)
                            : '\u2014'}
                        </TableCell>
                        <TableCell className="hidden md:table-cell">
                          {child.contract_from
                            ? `${fmt.date(child.contract_from)} \u2014 ${
                                child.contract_to ? fmt.date(child.contract_to) : t('ongoing')
                              }`
                            : '\u2014'}
                        </TableCell>
                        <TableCell>
                          {child.bill_appearances && child.bill_appearances.length > 0 ? (
                            <span className="text-sm">
                              {child.bill_appearances.map((a, i) => (
                                <span key={a.bill_id}>
                                  {i > 0 && ', '}
                                  <Link
                                    href={`/organizations/${orgId}/government-funding-bills/${a.bill_id}`}
                                    className="hover:text-primary hover:underline"
                                  >
                                    {fmt.monthYear(a.bill_from)}
                                  </Link>
                                </span>
                              ))}
                            </span>
                          ) : (
                            <span className="text-muted-foreground text-sm">
                              {t('neverInBill')}
                            </span>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          );
        })()}
    </div>
  );
}
