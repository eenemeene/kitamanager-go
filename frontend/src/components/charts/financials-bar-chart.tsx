'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';
import type { BarDatum, BarCustomLayerProps } from '@nivo/bar';
import type { FinancialResponse } from '@/lib/api/types';
import { buildKitaYearBands, formatDateLabel, chartTheme } from './chart-utils';

interface FinancialsBarChartProps {
  data: FinancialResponse;
}

function centsToEur(cents: number): number {
  return Math.round(cents) / 100;
}

export function FinancialsBarChart({ data }: FinancialsBarChartProps) {
  const t = useTranslations();

  const fundingKey = t('statistics.fundingIncome');
  const budgetIncomeKey = t('statistics.budgetIncome');
  const grossSalaryKey = t('statistics.grossSalary');
  const employerCostsKey = t('statistics.employerCosts');
  const budgetExpensesKey = t('statistics.budgetExpenses');

  const rawDates = data.data_points.map((dp) => dp.date);
  const xLabels = rawDates.map(formatDateLabel);
  const kitaYearBands = useMemo(() => buildKitaYearBands(rawDates), [rawDates]);

  const chartData: BarDatum[] = data.data_points.map((dp) => ({
    date: formatDateLabel(dp.date),
    [fundingKey]: centsToEur(dp.funding_income),
    [budgetIncomeKey]: centsToEur(dp.budget_income),
    [grossSalaryKey]: -centsToEur(dp.gross_salary),
    [employerCostsKey]: -centsToEur(dp.employer_costs),
    [budgetExpensesKey]: -centsToEur(dp.budget_expenses),
  }));

  const keys = [fundingKey, budgetIncomeKey, grossSalaryKey, employerCostsKey, budgetExpensesKey];
  const colors = ['#22c55e', '#14b8a6', '#ef4444', '#f97316', '#f59e0b'];

  const KitaYearBackground = useMemo(() => {
    return function KitaYearBg({ xScale, innerHeight, innerWidth }: BarCustomLayerProps<BarDatum>) {
      const scale = xScale as unknown as ((v: string) => number | undefined) & {
        bandwidth(): number;
      };
      const bw = scale.bandwidth();

      return (
        <g>
          {kitaYearBands.map((band, i) => {
            const x0 = scale(xLabels[band.startIdx]) ?? 0;
            const x1 = (scale(xLabels[band.endIdx]) ?? 0) + bw;
            const clampedX0 = Math.max(0, x0);
            const clampedX1 = Math.min(innerWidth, x1);
            const width = clampedX1 - clampedX0;

            return (
              <g key={band.label}>
                {i % 2 === 1 && (
                  <rect
                    x={clampedX0}
                    y={0}
                    width={width}
                    height={innerHeight}
                    fill="currentColor"
                    opacity={0.04}
                  />
                )}
                <text
                  x={clampedX0 + width / 2}
                  y={10}
                  textAnchor="middle"
                  fontSize={10}
                  fill="currentColor"
                  opacity={0.35}
                >
                  {t('statistics.kitaYear', { year: band.label })}
                </text>
              </g>
            );
          })}
        </g>
      );
    };
  }, [kitaYearBands, xLabels, t]);

  return (
    <div className="h-[350px]">
      <ResponsiveBar
        data={chartData}
        keys={keys}
        indexBy="date"
        margin={{ top: 40, right: 30, bottom: 50, left: 80 }}
        padding={0.3}
        groupMode="stacked"
        minValue="auto"
        maxValue="auto"
        colors={colors}
        layers={[KitaYearBackground, 'grid', 'axes', 'bars', 'markers', 'legends', 'annotations']}
        axisTop={null}
        axisRight={null}
        axisBottom={{
          tickSize: 5,
          tickPadding: 5,
          tickRotation: -45,
        }}
        axisLeft={{
          tickSize: 5,
          tickPadding: 5,
          tickRotation: 0,
          format: (v) =>
            Number(v).toLocaleString('de-DE', {
              style: 'currency',
              currency: 'EUR',
              maximumFractionDigits: 0,
            }),
        }}
        enableLabel={false}
        tooltip={({ id, value, indexValue, color }) => (
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
            <strong>{indexValue}</strong>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
              <span
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: '50%',
                  background: color,
                  display: 'inline-block',
                }}
              />
              {id}:{' '}
              {Math.abs(Number(value)).toLocaleString('de-DE', {
                style: 'currency',
                currency: 'EUR',
              })}
            </div>
          </div>
        )}
        legends={[
          {
            dataFrom: 'keys',
            anchor: 'top',
            direction: 'row',
            justify: false,
            translateX: 0,
            translateY: -35,
            itemsSpacing: 4,
            itemDirection: 'left-to-right',
            itemWidth: 130,
            itemHeight: 20,
            itemOpacity: 0.85,
            symbolSize: 12,
            symbolShape: 'circle',
          },
        ]}
        role="application"
        ariaLabel={t('statistics.financialBreakdown')}
        theme={chartTheme}
      />
    </div>
  );
}
