'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsivePie } from '@nivo/pie';
import type { FinancialDataPoint, FinancialSalaryDetail } from '@/lib/api/types';
import { chartTheme } from './chart-utils';
import { ExportableChart } from './exportable-chart';
import { useFormatters } from '@/hooks/use-formatters';

interface ExpenseBreakdownChartProps {
  data: FinancialDataPoint;
}

export interface SliceDatum {
  id: string;
  label: string;
  value: number;
  color: string;
  salaryDetail?: FinancialSalaryDetail;
}

export const EXPENSE_BREAKDOWN_COLORS = [
  '#ef4444',
  '#f97316',
  '#f59e0b',
  '#e879f9',
  '#fb923c',
  '#a855f7',
] as const;

/**
 * Build pie slices for the expense breakdown.
 *
 * Slice ordering (matters for color cycling and stable visual diffs):
 *   1. If `salary_details` is non-empty, one slice per category whose
 *      (gross + employer) total is > 0, in input order.
 *   2. Otherwise, fall back to aggregate salary: a `gross_salary` slice
 *      and/or an `employer_costs` slice, each only if > 0.
 *   3. Then expense-category budget items with `amount_cents > 0`,
 *      filtered by `category === 'expense'`, in input order.
 *
 * Values are converted from cents to euros; consumers pass `value * 100`
 * back into Intl currency formatting at the tooltip.
 *
 * Pure function, exported so unit tests can pin every branch without
 * having to render Nivo.
 */
export function buildExpenseSlices(
  data: FinancialDataPoint,
  t: (key: string) => string,
  colors: readonly string[] = EXPENSE_BREAKDOWN_COLORS
): SliceDatum[] {
  const slices: SliceDatum[] = [];
  let colorIdx = 0;

  if (data.salary_details?.length) {
    // Per-category salary slices (gross + employer combined per category)
    data.salary_details.forEach((sd) => {
      const total = (sd.gross_salary ?? 0) + (sd.employer_costs ?? 0);
      if (total > 0) {
        slices.push({
          id: `salary_${sd.staff_category}`,
          label: t(`employees.staffCategory.${sd.staff_category}`),
          value: total / 100,
          color: colors[colorIdx++ % colors.length]!,
          salaryDetail: sd,
        });
      }
    });
  } else {
    // Fallback: aggregate salary slices
    if ((data.gross_salary ?? 0) > 0) {
      slices.push({
        id: 'gross_salary',
        label: t('statistics.grossSalary'),
        value: (data.gross_salary ?? 0) / 100,
        color: colors[colorIdx++ % colors.length]!,
      });
    }

    if ((data.employer_costs ?? 0) > 0) {
      slices.push({
        id: 'employer_costs',
        label: t('statistics.employerCosts'),
        value: (data.employer_costs ?? 0) / 100,
        color: colors[colorIdx++ % colors.length]!,
      });
    }
  }

  data.budget_item_details
    ?.filter((bi) => bi.category === 'expense' && (bi.amount_cents ?? 0) > 0)
    .forEach((bi) => {
      slices.push({
        id: `budget_${bi.name}`,
        label: bi.name ?? '',
        value: (bi.amount_cents ?? 0) / 100,
        color: colors[colorIdx++ % colors.length]!,
      });
    });

  return slices;
}

export function ExpenseBreakdownChart({ data }: ExpenseBreakdownChartProps) {
  const t = useTranslations();

  const fmt = useFormatters();
  const formatEur = (cents: number) => fmt.currency(cents);
  const formatPct = (value: number, total: number) =>
    total === 0 ? '0%' : fmt.percentage(value / total, 1, true);
  const pieData = useMemo(() => buildExpenseSlices(data, t), [data, t]);

  const total = useMemo(() => pieData.reduce((sum, s) => sum + s.value, 0), [pieData]);

  if (pieData.length === 0) {
    return <p className="text-muted-foreground">{t('statistics.chartError')}</p>;
  }

  return (
    <ExportableChart filename="expense-breakdown" className="h-[350px]">
      {/* @nivo/pie takes no ariaLabel, so the name lives on a wrapper.
          It sits inside ExportableChart rather than on it, to keep the
          export button out of the image’s subtree. */}
      <div role="img" aria-label={t('statistics.expenseBreakdown')} className="h-full w-full">
        <ResponsivePie
          data={pieData}
          margin={{ top: 30, right: 120, bottom: 30, left: 120 }}
          innerRadius={0.5}
          padAngle={1}
          cornerRadius={3}
          activeOuterRadiusOffset={6}
          colors={{ datum: 'data.color' }}
          arcLinkLabel="label"
          arcLinkLabelsSkipAngle={10}
          arcLinkLabelsTextColor="hsl(var(--foreground))"
          arcLinkLabelsThickness={2}
          arcLinkLabelsColor={{ from: 'color' }}
          arcLabelsSkipAngle={10}
          arcLabelsTextColor="white"
          arcLabel={(d) => formatPct(d.value, total)}
          tooltip={({ datum }) => {
            const sd = (datum.data as SliceDatum).salaryDetail;
            return (
              <div
                style={{
                  background: 'hsl(var(--background))',
                  color: 'hsl(var(--foreground))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '6px',
                  padding: '9px 12px',
                  fontSize: 13,
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span
                    style={{
                      width: 10,
                      height: 10,
                      borderRadius: '50%',
                      background: datum.color,
                      display: 'inline-block',
                    }}
                  />
                  <strong>{datum.label}</strong>
                </div>
                <div style={{ marginTop: 4 }}>
                  {formatEur(datum.value * 100)} ({formatPct(datum.value, total)})
                </div>
                {sd && (
                  <div style={{ marginTop: 4, fontSize: 12, opacity: 0.8 }}>
                    <div>
                      {t('statistics.grossSalary')}: {formatEur(sd.gross_salary ?? 0)}
                    </div>
                    <div>
                      {t('statistics.employerCosts')}: {formatEur(sd.employer_costs ?? 0)}
                    </div>
                  </div>
                )}
              </div>
            );
          }}
          theme={chartTheme}
        />
      </div>
    </ExportableChart>
  );
}
