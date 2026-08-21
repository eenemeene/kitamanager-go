'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { HeaderWithTooltip } from '@/components/ui/header-with-tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { TooltipProvider } from '@/components/ui/tooltip';
import type { FinancialResponse } from '@/lib/api/types';
import { useFormatters } from '@/hooks/use-formatters';

interface BudgetTableProps {
  data: FinancialResponse;
}

export function BudgetTable({ data }: BudgetTableProps) {
  const t = useTranslations('statistics');
  const fmt = useFormatters();

  // Both of these were module-level helpers with 'de-DE' baked in. They live
  // in the component now because the locale does, and a zero renders as a dash
  // rather than "0,00 \u20ac" so an empty month reads as empty.
  const monthHeader = (dateStr: string) => fmt.monthYear(dateStr);
  const currencyCell = (cents: number) => (cents === 0 ? '\u2013' : fmt.currency(cents));

  // Memoised so the `?? []` fallback does not produce a fresh array on every
  // render, which would invalidate the useMemo hooks below that depend on it.
  const dataPoints = useMemo(() => data.data_points ?? [], [data.data_points]);

  // Extract unique budget item names by category across all data points
  const { incomeItems, expenseItems } = useMemo(() => {
    const incomeSet = new Set<string>();
    const expenseSet = new Set<string>();
    for (const dp of dataPoints) {
      for (const item of dp.budget_item_details ?? []) {
        if (item.category === 'income') {
          incomeSet.add(item.name ?? '');
        } else {
          expenseSet.add(item.name ?? '');
        }
      }
    }
    return {
      incomeItems: Array.from(incomeSet).sort(),
      expenseItems: Array.from(expenseSet).sort(),
    };
  }, [dataPoints]);

  const hasActualFunding = dataPoints.some((dp) => dp.actual_funding != null);

  // Build per-month row data
  const rows = useMemo(() => {
    return dataPoints.map((dp) => {
      const budgetMap = new Map<string, number>();
      for (const item of dp.budget_item_details ?? []) {
        budgetMap.set(item.name ?? '', item.amount_cents ?? 0);
      }
      return {
        date: dp.date ?? '',
        fundingIncome: dp.funding_income ?? 0,
        actualFunding: dp.actual_funding ?? null,
        incomeItemValues: incomeItems.map((name) => budgetMap.get(name) ?? 0),
        totalIncome: dp.total_income ?? 0,
        salaries: (dp.gross_salary ?? 0) + (dp.employer_costs ?? 0),
        expenseItemValues: expenseItems.map((name) => budgetMap.get(name) ?? 0),
        totalExpenses: dp.total_expenses ?? 0,
        balance: dp.balance ?? 0,
      };
    });
  }, [dataPoints, incomeItems, expenseItems]);

  // Compute totals row
  const totals = useMemo(() => {
    const sum = {
      fundingIncome: 0,
      actualFunding: null as number | null,
      incomeItemValues: incomeItems.map(() => 0),
      totalIncome: 0,
      salaries: 0,
      expenseItemValues: expenseItems.map(() => 0),
      totalExpenses: 0,
      balance: 0,
    };
    for (const row of rows) {
      sum.fundingIncome += row.fundingIncome;
      if (row.actualFunding != null) {
        sum.actualFunding = (sum.actualFunding ?? 0) + row.actualFunding;
      }
      for (let i = 0; i < incomeItems.length; i++) {
        sum.incomeItemValues[i] += row.incomeItemValues[i];
      }
      sum.totalIncome += row.totalIncome;
      sum.salaries += row.salaries;
      for (let i = 0; i < expenseItems.length; i++) {
        sum.expenseItemValues[i] += row.expenseItemValues[i];
      }
      sum.totalExpenses += row.totalExpenses;
      sum.balance += row.balance;
    }
    return sum;
  }, [rows, incomeItems, expenseItems]);

  // Column counts for header spans
  const incomeColCount = 1 + (hasActualFunding ? 1 : 0) + incomeItems.length + 1; // funding (calc) + funding (actual) + items + subtotal
  const expenseColCount = 1 + expenseItems.length + 1; // salaries + items + subtotal

  if (dataPoints.length === 0) {
    return <p className="text-muted-foreground">{t('chartError')}</p>;
  }

  return (
    <TooltipProvider>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            {/* Group header row */}
            <TableRow>
              <TableHead className="bg-background sticky left-0 z-10" rowSpan={2} />
              <TableHead colSpan={incomeColCount} className="text-success text-center">
                {t('totalIncome')}
              </TableHead>
              <TableHead colSpan={expenseColCount} className="text-destructive text-center">
                {t('totalExpenses')}
              </TableHead>
              <TableHead rowSpan={2} className="text-center">
                {t('balance')}
              </TableHead>
            </TableRow>
            {/* Sub-header row */}
            <TableRow>
              {/* Income sub-headers */}
              <TableHead className="text-center">
                <HeaderWithTooltip
                  label={t('fundingIncomeSub')}
                  tooltip={t('fundingIncomeSubTooltip')}
                />
              </TableHead>
              {hasActualFunding && (
                <TableHead className="text-center">
                  <HeaderWithTooltip
                    label={t('fundingActualSub')}
                    tooltip={t('fundingActualSubTooltip')}
                  />
                </TableHead>
              )}
              {incomeItems.map((name) => (
                <TableHead key={name} className="text-center">
                  {name}
                </TableHead>
              ))}
              <TableHead className="text-center font-bold">{t('incomeTotal')}</TableHead>
              {/* Expense sub-headers */}
              <TableHead className="text-center">{t('salaries')}</TableHead>
              {expenseItems.map((name) => (
                <TableHead key={name} className="text-center">
                  {name}
                </TableHead>
              ))}
              <TableHead className="text-center font-bold">{t('expenseTotal')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {/* Monthly rows */}
            {rows.map((row) => (
              <TableRow key={row.date}>
                <TableCell className="bg-background sticky left-0 z-10 font-medium">
                  {monthHeader(row.date)}
                </TableCell>
                {/* Income columns */}
                <TableCell className="text-right tabular-nums">
                  {currencyCell(row.fundingIncome)}
                </TableCell>
                {hasActualFunding && (
                  <TableCell className="text-right tabular-nums">
                    {row.actualFunding != null ? currencyCell(row.actualFunding) : '\u2013'}
                  </TableCell>
                )}
                {row.incomeItemValues.map((val, i) => (
                  <TableCell key={incomeItems[i]} className="text-right tabular-nums">
                    {currencyCell(val)}
                  </TableCell>
                ))}
                <TableCell className="text-success text-right font-bold tabular-nums">
                  {currencyCell(row.totalIncome)}
                </TableCell>
                {/* Expense columns */}
                <TableCell className="text-right tabular-nums">
                  {currencyCell(row.salaries)}
                </TableCell>
                {row.expenseItemValues.map((val, i) => (
                  <TableCell key={expenseItems[i]} className="text-right tabular-nums">
                    {currencyCell(val)}
                  </TableCell>
                ))}
                <TableCell className="text-destructive text-right font-bold tabular-nums">
                  {currencyCell(row.totalExpenses)}
                </TableCell>
                {/* Balance */}
                <TableCell
                  className={`text-right font-bold tabular-nums ${
                    row.balance >= 0 ? 'text-success' : 'text-destructive'
                  }`}
                >
                  {currencyCell(row.balance)}
                </TableCell>
              </TableRow>
            ))}

            {/* Annual total row */}
            <TableRow className="border-t-2 font-bold">
              <TableCell className="bg-background sticky left-0 z-10">{t('annualTotal')}</TableCell>
              <TableCell className="text-right tabular-nums">
                {currencyCell(totals.fundingIncome)}
              </TableCell>
              {hasActualFunding && (
                <TableCell className="text-right tabular-nums">
                  {totals.actualFunding != null ? currencyCell(totals.actualFunding) : '\u2013'}
                </TableCell>
              )}
              {totals.incomeItemValues.map((val, i) => (
                <TableCell key={incomeItems[i]} className="text-right tabular-nums">
                  {currencyCell(val)}
                </TableCell>
              ))}
              <TableCell className="text-success text-right tabular-nums">
                {currencyCell(totals.totalIncome)}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {currencyCell(totals.salaries)}
              </TableCell>
              {totals.expenseItemValues.map((val, i) => (
                <TableCell key={expenseItems[i]} className="text-right tabular-nums">
                  {currencyCell(val)}
                </TableCell>
              ))}
              <TableCell className="text-destructive text-right tabular-nums">
                {currencyCell(totals.totalExpenses)}
              </TableCell>
              <TableCell
                className={`text-right tabular-nums ${
                  totals.balance >= 0 ? 'text-success' : 'text-destructive'
                }`}
              >
                {currencyCell(totals.balance)}
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </TooltipProvider>
  );
}
