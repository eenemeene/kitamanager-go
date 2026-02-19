'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { FinancialResponse } from '@/lib/api/types';
import { formatCurrency } from '@/lib/utils/formatting';

interface BudgetTableProps {
  data: FinancialResponse;
}

function formatMonthHeader(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  return date.toLocaleDateString('de-DE', { month: 'short', year: '2-digit' });
}

function formatCurrencyCell(cents: number): string {
  if (cents === 0) return '\u2013';
  return formatCurrency(cents);
}

export function BudgetTable({ data }: BudgetTableProps) {
  const t = useTranslations('statistics');

  const dataPoints = data.data_points ?? [];

  // Extract unique budget item names by category across all data points
  const { incomeItems, expenseItems } = useMemo(() => {
    const incomeSet = new Set<string>();
    const expenseSet = new Set<string>();
    for (const dp of dataPoints) {
      for (const item of dp.budget_item_details ?? []) {
        if (item.category === 'income') {
          incomeSet.add(item.name);
        } else {
          expenseSet.add(item.name);
        }
      }
    }
    return {
      incomeItems: Array.from(incomeSet).sort(),
      expenseItems: Array.from(expenseSet).sort(),
    };
  }, [dataPoints]);

  // Build per-month row data
  const rows = useMemo(() => {
    return dataPoints.map((dp) => {
      const budgetMap = new Map<string, number>();
      for (const item of dp.budget_item_details ?? []) {
        budgetMap.set(item.name, item.amount_cents);
      }
      return {
        date: dp.date,
        fundingIncome: dp.funding_income,
        incomeItemValues: incomeItems.map((name) => budgetMap.get(name) ?? 0),
        totalIncome: dp.total_income,
        salaries: dp.gross_salary + dp.employer_costs,
        expenseItemValues: expenseItems.map((name) => budgetMap.get(name) ?? 0),
        totalExpenses: dp.total_expenses,
        balance: dp.balance,
      };
    });
  }, [dataPoints, incomeItems, expenseItems]);

  // Compute totals row
  const totals = useMemo(() => {
    const sum = {
      fundingIncome: 0,
      incomeItemValues: incomeItems.map(() => 0),
      totalIncome: 0,
      salaries: 0,
      expenseItemValues: expenseItems.map(() => 0),
      totalExpenses: 0,
      balance: 0,
    };
    for (const row of rows) {
      sum.fundingIncome += row.fundingIncome;
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
  const incomeColCount = 1 + incomeItems.length + 1; // funding + items + subtotal
  const expenseColCount = 1 + expenseItems.length + 1; // salaries + items + subtotal

  if (dataPoints.length === 0) {
    return <p className="text-muted-foreground">{t('chartError')}</p>;
  }

  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          {/* Group header row */}
          <TableRow>
            <TableHead className="bg-background sticky left-0 z-10" rowSpan={2} />
            <TableHead
              colSpan={incomeColCount}
              className="text-center text-green-700 dark:text-green-400"
            >
              {t('totalIncome')}
            </TableHead>
            <TableHead
              colSpan={expenseColCount}
              className="text-center text-red-700 dark:text-red-400"
            >
              {t('totalExpenses')}
            </TableHead>
            <TableHead rowSpan={2} className="text-center">
              {t('balance')}
            </TableHead>
          </TableRow>
          {/* Sub-header row */}
          <TableRow>
            {/* Income sub-headers */}
            <TableHead className="text-center">{t('fundingIncomeSub')}</TableHead>
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
                {formatMonthHeader(row.date)}
              </TableCell>
              {/* Income columns */}
              <TableCell className="text-right tabular-nums">
                {formatCurrencyCell(row.fundingIncome)}
              </TableCell>
              {row.incomeItemValues.map((val, i) => (
                <TableCell key={incomeItems[i]} className="text-right tabular-nums">
                  {formatCurrencyCell(val)}
                </TableCell>
              ))}
              <TableCell className="text-right font-bold text-green-700 tabular-nums dark:text-green-400">
                {formatCurrencyCell(row.totalIncome)}
              </TableCell>
              {/* Expense columns */}
              <TableCell className="text-right tabular-nums">
                {formatCurrencyCell(row.salaries)}
              </TableCell>
              {row.expenseItemValues.map((val, i) => (
                <TableCell key={expenseItems[i]} className="text-right tabular-nums">
                  {formatCurrencyCell(val)}
                </TableCell>
              ))}
              <TableCell className="text-right font-bold text-red-700 tabular-nums dark:text-red-400">
                {formatCurrencyCell(row.totalExpenses)}
              </TableCell>
              {/* Balance */}
              <TableCell
                className={`text-right font-bold tabular-nums ${
                  row.balance >= 0
                    ? 'text-green-700 dark:text-green-400'
                    : 'text-red-700 dark:text-red-400'
                }`}
              >
                {formatCurrencyCell(row.balance)}
              </TableCell>
            </TableRow>
          ))}

          {/* Annual total row */}
          <TableRow className="border-t-2 font-bold">
            <TableCell className="bg-background sticky left-0 z-10">{t('annualTotal')}</TableCell>
            <TableCell className="text-right tabular-nums">
              {formatCurrencyCell(totals.fundingIncome)}
            </TableCell>
            {totals.incomeItemValues.map((val, i) => (
              <TableCell key={incomeItems[i]} className="text-right tabular-nums">
                {formatCurrencyCell(val)}
              </TableCell>
            ))}
            <TableCell className="text-right text-green-700 tabular-nums dark:text-green-400">
              {formatCurrencyCell(totals.totalIncome)}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {formatCurrencyCell(totals.salaries)}
            </TableCell>
            {totals.expenseItemValues.map((val, i) => (
              <TableCell key={expenseItems[i]} className="text-right tabular-nums">
                {formatCurrencyCell(val)}
              </TableCell>
            ))}
            <TableCell className="text-right text-red-700 tabular-nums dark:text-red-400">
              {formatCurrencyCell(totals.totalExpenses)}
            </TableCell>
            <TableCell
              className={`text-right tabular-nums ${
                totals.balance >= 0
                  ? 'text-green-700 dark:text-green-400'
                  : 'text-red-700 dark:text-red-400'
              }`}
            >
              {formatCurrencyCell(totals.balance)}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}
