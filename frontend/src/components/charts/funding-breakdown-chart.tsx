'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsivePie } from '@nivo/pie';
import type { FinancialDataPoint } from '@/lib/api/types';
import { chartTheme } from './chart-utils';
import { ExportableChart } from './exportable-chart';
import { useFormatters } from '@/hooks/use-formatters';

interface FundingBreakdownChartProps {
  data: FinancialDataPoint;
}

export const FUNDING_BREAKDOWN_COLORS = [
  '#22c55e',
  '#14b8a6',
  '#06b6d4',
  '#8b5cf6',
  '#f59e0b',
  '#ec4899',
] as const;

export interface FundingSliceDatum {
  id: string;
  label: string;
  value: number;
  color: string;
}

/**
 * Build pie slices for the funding breakdown.
 *
 * Slice ordering (matters for color cycling and legend stability):
 *   1. Government funding entries (`funding_details`) with
 *      `amount_cents > 0`, in input order.
 *   2. Then income-category budget items (`budget_item_details`) with
 *      `amount_cents > 0`, in input order.
 *
 * Values are converted from cents to euros. Pure function exported so
 * unit tests can pin every branch without rendering Nivo.
 */
export function buildFundingSlices(
  data: FinancialDataPoint,
  colors: readonly string[] = FUNDING_BREAKDOWN_COLORS
): FundingSliceDatum[] {
  const slices: FundingSliceDatum[] = [];
  let colorIdx = 0;

  data.funding_details?.forEach((fd) => {
    if ((fd.amount_cents ?? 0) > 0) {
      slices.push({
        id: `funding_${fd.key}_${fd.value}`,
        label: fd.label ?? '',
        value: (fd.amount_cents ?? 0) / 100,
        color: colors[colorIdx++ % colors.length]!,
      });
    }
  });

  data.budget_item_details
    ?.filter((bi) => bi.category === 'income' && (bi.amount_cents ?? 0) > 0)
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

export function FundingBreakdownChart({ data }: FundingBreakdownChartProps) {
  const t = useTranslations();

  const fmt = useFormatters();
  const formatEur = (cents: number) => fmt.currency(cents);
  const formatPct = (value: number, total: number) =>
    total === 0 ? '0%' : fmt.percentage(value / total, 1, true);
  const pieData = useMemo(() => buildFundingSlices(data), [data]);

  const total = useMemo(() => pieData.reduce((sum, s) => sum + s.value, 0), [pieData]);

  if (pieData.length === 0) {
    return <p className="text-muted-foreground">{t('statistics.chartError')}</p>;
  }

  return (
    <ExportableChart filename="funding-breakdown" className="h-[350px]">
      {/* @nivo/pie takes no ariaLabel, so the name lives on a wrapper.
          It sits inside ExportableChart rather than on it, to keep the
          export button out of the image’s subtree. */}
      <div role="img" aria-label={t('statistics.fundingBreakdown')} className="h-full w-full">
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
          tooltip={({ datum }) => (
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
            </div>
          )}
          theme={chartTheme}
        />
      </div>
    </ExportableChart>
  );
}
