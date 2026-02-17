'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsivePie } from '@nivo/pie';
import type { FinancialDataPoint } from '@/lib/api/types';
import { chartTheme } from './chart-utils';

interface FundingBreakdownChartProps {
  data: FinancialDataPoint;
}

function formatEur(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

function formatPct(value: number, total: number): string {
  if (total === 0) return '0%';
  return `${((value / total) * 100).toFixed(1)}%`;
}

const COLORS = ['#22c55e', '#14b8a6', '#06b6d4', '#8b5cf6', '#f59e0b', '#ec4899'];

export function FundingBreakdownChart({ data }: FundingBreakdownChartProps) {
  const t = useTranslations();

  const pieData = useMemo(() => {
    const slices: { id: string; label: string; value: number; color: string }[] = [];
    let colorIdx = 0;

    data.funding_details?.forEach((fd) => {
      if (fd.amount_cents > 0) {
        slices.push({
          id: `funding_${fd.key}_${fd.value}`,
          label: fd.label,
          value: fd.amount_cents / 100,
          color: COLORS[colorIdx++ % COLORS.length],
        });
      }
    });

    data.budget_item_details
      ?.filter((bi) => bi.category === 'income' && bi.amount_cents > 0)
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

  const total = useMemo(() => pieData.reduce((sum, s) => sum + s.value, 0), [pieData]);

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
  );
}
