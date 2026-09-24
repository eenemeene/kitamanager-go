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

/**
 * The entries `buildFundingSlices` cannot draw.
 *
 * A pie shows parts of a whole, and a deduction is not one — there is no
 * negative slice. The filter that keeps them out was right; leaving them
 * unsaid was not. In Berlin the parent meal contribution applies to *every*
 * contract (`apply_to_all_contracts`), so for a fifty-child Kita the chart was
 * quietly 1.150,00 EUR short of the income it claimed to break down, and every
 * percentage was computed against that inflated base.
 *
 * Only funding details can be negative. Budget item amounts are non-negative by
 * construction (`binding:"required,min=0"`, and the model says "cents, always
 * positive"), so they are not searched here.
 */
export function buildFundingDeductions(data: FinancialDataPoint): FundingSliceDatum[] {
  return (data.funding_details ?? [])
    .filter((fd) => (fd.amount_cents ?? 0) < 0)
    .map((fd) => ({
      id: `deduction_${fd.key}_${fd.value}`,
      label: fd.label ?? '',
      value: (fd.amount_cents ?? 0) / 100,
      color: 'currentColor',
    }));
}

/** Draws the net total in the donut's hole. */
function makeCenterMetric(amount: string, caption: string) {
  return function CenterMetric({ centerX, centerY }: { centerX: number; centerY: number }) {
    return (
      <g>
        <text
          x={centerX}
          y={centerY - 4}
          textAnchor="middle"
          dominantBaseline="central"
          style={{ fontSize: 18, fontWeight: 600, fill: 'hsl(var(--foreground))' }}
        >
          {amount}
        </text>
        <text
          x={centerX}
          y={centerY + 14}
          textAnchor="middle"
          dominantBaseline="central"
          style={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
        >
          {caption}
        </text>
      </g>
    );
  };
}

export function FundingBreakdownChart({ data }: FundingBreakdownChartProps) {
  const t = useTranslations();

  const fmt = useFormatters();
  const formatEur = (cents: number) => fmt.currency(cents);
  const formatPct = (value: number, total: number) =>
    total === 0 ? '0%' : fmt.percentage(value / total, 1, true);
  const pieData = useMemo(() => buildFundingSlices(data), [data]);
  const deductions = useMemo(() => buildFundingDeductions(data), [data]);

  // Gross is what the slices add up to and what the percentages are shares of.
  // Net is the income the rest of the page reports, so the two have to be shown
  // together or the chart silently contradicts the summary card beside it.
  const total = useMemo(() => pieData.reduce((sum, s) => sum + s.value, 0), [pieData]);
  const deducted = useMemo(() => deductions.reduce((sum, d) => sum + d.value, 0), [deductions]);
  const net = total + deducted; // deducted is negative
  const CenterMetric = useMemo(
    () => makeCenterMetric(formatEur(net * 100), t('statistics.totalIncome')),
    // formatEur closes over the locale formatter, which is stable per render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [net, t]
  );

  if (pieData.length === 0) {
    // Nothing to draw. If the month is made of deductions alone the figure is
    // still worth stating -- an empty chart where money moved reads as "no
    // data", which is a different and wrong answer.
    if (deductions.length === 0) {
      return <p className="text-muted-foreground">{t('statistics.chartError')}</p>;
    }
    return (
      <p data-visual-mask="currency" className="text-muted-foreground text-sm">
        {t('statistics.fundingBreakdownNet', { amount: formatEur(net * 100) })}
      </p>
    );
  }

  const chart = (
    <ExportableChart filename="funding-breakdown" className="h-[350px]">
      {/* @nivo/pie takes no ariaLabel, so the name lives on a wrapper.
          It sits inside ExportableChart rather than on it, to keep the
          export button out of the image’s subtree. */}
      <div
        role="img"
        aria-label={t('statistics.fundingBreakdown')}
        className="relative h-full w-full"
      >
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
          // An SVG layer rather than an HTML overlay: the export button writes
          // out the SVG, and a shared image showing shares of a total it does
          // not state is the same omission this fix is about.
          layers={['arcs', 'arcLinkLabels', 'arcLabels', 'legends', CenterMetric]}
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

  if (deductions.length === 0) {
    return chart;
  }

  return (
    <div>
      {chart}
      {/* Outside the chart wrapper on purpose: that wrapper is a fixed 350px
          box, so a sibling paragraph inside it would overflow. Carries its own
          mask because the figures move with the data. */}
      <p data-visual-mask="currency" className="text-muted-foreground mt-1 text-center text-xs">
        {t('statistics.fundingBreakdownReconcile', {
          gross: formatEur(total * 100),
          deductions: deductions.map((d) => d.label).join(', '),
          deducted: formatEur(-deducted * 100),
          net: formatEur(net * 100),
        })}
      </p>
    </div>
  );
}
