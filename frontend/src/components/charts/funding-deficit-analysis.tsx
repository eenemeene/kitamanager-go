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

interface FundingDeficitAnalysisProps {
  summary: FundingComparisonSummary;
  orgId: string | string[];
  /** When true, the deficit analysis section is expanded. Used for print/PDF export. */
  forceExpanded?: boolean;
}

function formatEur(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

const MAX_INITIAL_ISSUES = 10;

export function FundingDeficitAnalysis({
  summary,
  orgId,
  forceExpanded = false,
}: FundingDeficitAnalysisProps) {
  const t = useTranslations('statistics');
  const [showAllIssues, setShowAllIssues] = useState(forceExpanded);
  const [expanded, setExpanded] = useState(forceExpanded);

  const { categories, issues } = summary;
  const actionableIssues = issues.filter((i) => i.actionable);
  const maxCategoryAbs = Math.max(...categories.map((c) => Math.abs(c.total_amount)), 1);

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
                  {categories.map((cat) => (
                    <div key={cat.category} className="flex items-center gap-3">
                      <span className="w-36 truncate text-sm">
                        {t(`deficitCategory_${cat.category}`)}
                      </span>
                      <div className="bg-muted h-2 flex-1 rounded-full">
                        <div
                          className={cn(
                            'h-2 rounded-full',
                            cat.total_amount < 0 ? 'bg-destructive/70' : 'bg-success/70'
                          )}
                          style={{
                            width: `${(Math.abs(cat.total_amount) / maxCategoryAbs) * 100}%`,
                          }}
                        />
                      </div>
                      <span
                        className={cn(
                          'w-24 text-right text-sm font-medium tabular-nums',
                          cat.total_amount < 0 ? 'text-destructive' : 'text-success'
                        )}
                      >
                        {cat.total_amount >= 0 ? '+' : ''}
                        {formatEur(cat.total_amount)}
                      </span>
                      <span className="text-muted-foreground w-16 text-right text-xs">
                        {t('fundingChildCount', { count: cat.child_count })}
                      </span>
                    </div>
                  ))}
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
                                    href={`/organizations/${orgId}/children/${issue.child_id}`}
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
                              {formatEur(issue.amount_per_month)} &times; {issue.month_count}m
                            </TableCell>
                            <TableCell
                              className={cn(
                                'text-right font-medium tabular-nums',
                                issue.total_amount < 0 ? 'text-destructive' : 'text-success'
                              )}
                            >
                              {issue.total_amount >= 0 ? '+' : ''}
                              {formatEur(issue.total_amount)}
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
