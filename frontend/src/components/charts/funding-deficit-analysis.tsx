'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { ChevronDown, ChevronRight } from 'lucide-react';
import type { FundingComparisonSummary } from '@/lib/api/types';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { useFormatters } from '@/hooks/use-formatters';

interface FundingDeficitAnalysisProps {
  summary: FundingComparisonSummary;
  orgId: string | string[];
  /** When true, the deficit analysis section is expanded. Used for print/PDF export. */
  forceExpanded?: boolean;
}

const MAX_INITIAL_ISSUES = 10;

export function FundingDeficitAnalysis({
  summary,
  orgId,
  forceExpanded = false,
}: FundingDeficitAnalysisProps) {
  const t = useTranslations('statistics');
  const fmt = useFormatters();
  const formatEur = (cents: number) => fmt.currency(cents);
  const [showAllIssues, setShowAllIssues] = useState(forceExpanded);
  const [expanded, setExpanded] = useState(forceExpanded);

  const categories = summary.categories ?? [];
  const issues = summary.issues ?? [];
  const actionableIssues = issues.filter((i) => i.actionable);
  const maxCategoryAbs = Math.max(...categories.map((c) => Math.abs(c.total_amount ?? 0)), 1);

  // The bars are an exhaustive decomposition of total_difference, which counts
  // regular billing only. The Kita year row above them also carries the
  // corrections, so without this the two never agreed and nothing on screen
  // said why.
  //
  // Corrections attributed to these months, not the ones that arrived in them:
  // a correction paid out in August against July belongs to July's Kita year.
  // Falls back to the arrival-keyed total for a single-bill comparison, which
  // has no window to attribute against.
  const categoriesSum = categories.reduce((acc, c) => acc + (c.total_amount ?? 0), 0);
  const corrections = summary.total_corrections_attributed ?? summary.total_corrections ?? 0;
  const reconciledTotal = categoriesSum + corrections;

  const visibleIssues = showAllIssues
    ? actionableIssues
    : actionableIssues.slice(0, MAX_INITIAL_ISSUES);

  if (categories.length === 0) {
    return null;
  }

  return (
    <TableRow className="bg-muted/10">
      <TableCell colSpan={5} className="px-6 py-3">
        <div className="space-y-3">
          <button
            type="button"
            className="flex items-center gap-2 text-sm font-semibold"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? (
              <ChevronDown className="h-4 w-4 shrink-0" />
            ) : (
              <ChevronRight className="h-4 w-4 shrink-0" />
            )}
            {t('deficitAnalysis')}
            {actionableIssues.length > 0 && (
              <Badge variant="outline" className="ml-2 text-xs">
                {t('deficitActionableCount', { count: actionableIssues.length })}
              </Badge>
            )}
          </button>

          {expanded && (
            <div className="space-y-4 pl-6">
              {/* Category breakdown */}
              <div className="space-y-2">
                <h5 className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                  {t('deficitCategories')}
                </h5>
                <div className="space-y-1.5">
                  {categories.map((cat) => {
                    const totalAmount = cat.total_amount ?? 0;
                    return (
                      <div key={cat.category} className="flex items-center gap-3">
                        <span className="w-36 truncate text-sm">
                          {t(`deficitCategory_${cat.category}`)}
                        </span>
                        <div className="bg-muted h-2 flex-1 rounded-full">
                          <div
                            className={cn(
                              'h-2 rounded-full',
                              totalAmount < 0 ? 'bg-destructive/70' : 'bg-success/70'
                            )}
                            style={{
                              width: `${(Math.abs(totalAmount) / maxCategoryAbs) * 100}%`,
                            }}
                          />
                        </div>
                        <span
                          className={cn(
                            'w-24 text-right text-sm font-medium tabular-nums',
                            totalAmount < 0 ? 'text-destructive' : 'text-success'
                          )}
                        >
                          {totalAmount >= 0 ? '+' : ''}
                          {formatEur(totalAmount)}
                        </span>
                        <span className="text-muted-foreground w-16 text-right text-xs">
                          {t('fundingChildCount', { count: cat.child_count ?? 0 })}
                        </span>
                      </div>
                    );
                  })}
                </div>

                {/* The arithmetic, spelled out. A reader who subtracts the
                    bars and compares the result with the row above needs to
                    see where the corrections enter, or the two figures look
                    like a contradiction. */}
                <div className="space-y-1 border-t pt-2">
                  <div className="text-muted-foreground flex items-center gap-3 text-xs">
                    <span className="w-36 truncate">{t('deficitCategoriesSum')}</span>
                    <div className="flex-1" />
                    <span data-visual-mask="currency" className="w-24 text-right tabular-nums">
                      {categoriesSum >= 0 ? '+' : ''}
                      {formatEur(categoriesSum)}
                    </span>
                    <span className="w-16" />
                  </div>
                  {corrections !== 0 && (
                    <div
                      className="text-muted-foreground flex items-center gap-3 text-xs"
                      title={t('deficitCorrectionsTooltip')}
                    >
                      <span className="w-36 truncate">{t('deficitCorrections')}</span>
                      <div className="flex-1" />
                      <span data-visual-mask="currency" className="w-24 text-right tabular-nums">
                        {corrections >= 0 ? '+' : ''}
                        {formatEur(corrections)}
                      </span>
                      <span className="w-16" />
                    </div>
                  )}
                  <div
                    className="flex items-center gap-3 text-sm font-medium"
                    title={t('deficitReconciledTotalTooltip')}
                  >
                    <span className="w-36 truncate">{t('deficitReconciledTotal')}</span>
                    <div className="flex-1" />
                    <span
                      data-visual-mask="currency"
                      className={cn(
                        'w-24 text-right tabular-nums',
                        reconciledTotal < 0 ? 'text-destructive' : 'text-success'
                      )}
                    >
                      {reconciledTotal >= 0 ? '+' : ''}
                      {formatEur(reconciledTotal)}
                    </span>
                    <span className="w-16" />
                  </div>
                </div>
              </div>

              {/* Actionable issues table */}
              {actionableIssues.length > 0 && (
                <div className="space-y-2">
                  <h5 className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                    {t('deficitIssues')}
                  </h5>
                  <div className="overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="text-xs">{t('deficitIssueChild')}</TableHead>
                          <TableHead className="hidden text-xs md:table-cell">
                            {t('deficitIssueDescription')}
                          </TableHead>
                          <TableHead className="text-right text-xs">
                            {t('deficitIssuePerMonth')}
                          </TableHead>
                          <TableHead className="text-right text-xs">
                            {t('deficitIssueAmount')}
                          </TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {visibleIssues.map((issue, idx) => (
                          <TableRow key={idx} className="text-sm">
                            <TableCell>
                              <div>
                                {issue.child_id ? (
                                  <Link
                                    href={`/organizations/${orgId}/children/${issue.child_id}/billing`}
                                    className="text-primary hover:underline"
                                  >
                                    {issue.child_name}
                                  </Link>
                                ) : (
                                  <span>{issue.child_name}</span>
                                )}
                              </div>
                              <div className="text-muted-foreground text-xs md:hidden">
                                {issue.description}
                              </div>
                            </TableCell>
                            <TableCell className="text-muted-foreground hidden text-xs md:table-cell">
                              {issue.description}
                            </TableCell>
                            <TableCell className="text-muted-foreground text-right text-xs tabular-nums">
                              {/* amount_per_month is total/months in integer cents,
                                  so it does not multiply back: -1000 over three
                                  months is -333, and -333 x 3 is -999. Marked as
                                  a mean so the row stops implying an identity it
                                  cannot satisfy. */}
                              &#8709; {formatEur(issue.amount_per_month ?? 0)} &middot;{' '}
                              {t('deficitIssueMonths', { count: issue.month_count ?? 0 })}
                            </TableCell>
                            <TableCell
                              className={cn(
                                'text-right font-medium tabular-nums',
                                (issue.total_amount ?? 0) < 0 ? 'text-destructive' : 'text-success'
                              )}
                            >
                              {(issue.total_amount ?? 0) >= 0 ? '+' : ''}
                              {formatEur(issue.total_amount ?? 0)}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                  {actionableIssues.length > MAX_INITIAL_ISSUES && (
                    <button
                      type="button"
                      className="text-primary text-xs hover:underline"
                      onClick={() => setShowAllIssues(!showAllIssues)}
                    >
                      {showAllIssues
                        ? t('deficitShowLess')
                        : t('deficitShowAll', { count: actionableIssues.length })}
                    </button>
                  )}
                </div>
              )}

              {actionableIssues.length === 0 && (
                <p className="text-muted-foreground text-sm">{t('deficitNoIssues')}</p>
              )}
            </div>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}
