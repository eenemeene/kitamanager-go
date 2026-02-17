'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsivePie } from '@nivo/pie';
import type { FinancialDataPoint, FinancialSalaryDetail } from '@/lib/api/types';
import { chartTheme } from './chart-utils';

interface ExpenseBreakdownChartProps {
  data: FinancialDataPoint;
}

interface SliceDatum {
  id: string;
  label: string;
  value: number;
  color: string;
  salaryDetail?: FinancialSalaryDetail;
}

function formatEur(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

const COLORS = ['#ef4444', '#f97316', '#f59e0b', '#e879f9', '#fb923c', '#a855f7'];

export function ExpenseBreakdownChart({ data }: ExpenseBreakdownChartProps) {
  const t = useTranslations();

  const pieData = useMemo(() => {
    const slices: SliceDatum[] = [];
    let colorIdx = 0;

    if (data.salary_details?.length) {
      // Per-category salary slices (gross + employer combined per category)
      data.salary_details.forEach((sd) => {
        const total = sd.gross_salary + sd.employer_costs;
        if (total > 0) {
          slices.push({
            id: `salary_${sd.staff_category}`,
            label: t(`employees.staffCategory.${sd.staff_category}`),
            value: total / 100,
            color: COLORS[colorIdx++ % COLORS.length],
            salaryDetail: sd,
          });
        }
      });
    } else {
      // Fallback: aggregate salary slices
      if (data.gross_salary > 0) {
        slices.push({
          id: 'gross_salary',
          label: t('statistics.grossSalary'),
          value: data.gross_salary / 100,
          color: COLORS[colorIdx++ % COLORS.length],
        });
      }

      if (data.employer_costs > 0) {
        slices.push({
          id: 'employer_costs',
          label: t('statistics.employerCosts'),
          value: data.employer_costs / 100,
          color: COLORS[colorIdx++ % COLORS.length],
        });
      }
    }

    data.budget_item_details
      ?.filter((bi) => bi.category === 'expense' && bi.amount_cents > 0)
      .forEach((bi) => {
        slices.push({
          id: `budget_${bi.name}`,
          label: bi.name,
          value: bi.amount_cents / 100,
          color: COLORS[colorIdx++ % COLORS.length],
        });
      });

    return slices;
  }, [data, t]);

  if (pieData.length === 0) {
    return <p className="text-muted-foreground">{t('statistics.chartError')}</p>;
  }

  return (
    <div className="h-[350px]">
      <ResponsivePie
        data={pieData}
        margin={{ top: 30, right: 120, bottom: 30, left: 120 }}
        innerRadius={0.5}
        padAngle={1}
        cornerRadius={3}
        activeOuterRadiusOffset={6}
        colors={{ datum: 'data.color' }}
        arcLinkLabelsSkipAngle={10}
        arcLinkLabelsTextColor="hsl(var(--foreground))"
        arcLinkLabelsThickness={2}
        arcLinkLabelsColor={{ from: 'color' }}
        arcLabelsSkipAngle={10}
        arcLabelsTextColor="white"
        valueFormat={(v) => formatEur(v * 100)}
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
              <div style={{ marginTop: 4 }}>{formatEur(datum.value * 100)}</div>
              {sd && (
                <div style={{ marginTop: 4, fontSize: 12, opacity: 0.8 }}>
                  <div>
                    {t('statistics.grossSalary')}: {formatEur(sd.gross_salary)}
                  </div>
                  <div>
                    {t('statistics.employerCosts')}: {formatEur(sd.employer_costs)}
                  </div>
                </div>
              )}
            </div>
          );
        }}
        theme={chartTheme}
      />
    </div>
  );
}
