'use client';

import { useMemo, useState } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveLine } from '@nivo/line';
import { ExportableChart } from './exportable-chart';
import { chartTheme } from './chart-utils';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { PayPlanPeriod } from '@/lib/api/types';
import { formatCurrency } from '@/lib/utils/formatting';

interface PayPlanSalaryChartProps {
  periods: PayPlanPeriod[];
}

/** A distinct color palette for up to 15 grade lines. */
const GRADE_COLORS = [
  '#3b82f6', // blue
  '#ef4444', // red
  '#22c55e', // green
  '#f59e0b', // amber
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#14b8a6', // teal
  '#f97316', // orange
  '#6366f1', // indigo
  '#06b6d4', // cyan
  '#84cc16', // lime
  '#d946ef', // fuchsia
  '#0ea5e9', // sky
  '#a855f7', // purple
  '#10b981', // emerald
];

export function PayPlanSalaryChart({ periods }: PayPlanSalaryChartProps) {
  const t = useTranslations();

  // Collect all unique steps and grades across periods
  const { allSteps, allGrades } = useMemo(() => {
    const stepSet = new Set<number>();
    const gradeSet = new Set<string>();
    for (const period of periods) {
      for (const entry of period.entries ?? []) {
        stepSet.add(entry.step);
        gradeSet.add(entry.grade);
      }
    }
    return {
      allSteps: Array.from(stepSet).sort((a, b) => a - b),
      allGrades: Array.from(gradeSet).sort(),
    };
  }, [periods]);

  const [selectedStep, setSelectedStep] = useState<number>(() => allSteps[0] ?? 1);

  // Sort periods chronologically
  const sortedPeriods = useMemo(
    () => [...periods].sort((a, b) => new Date(a.from).getTime() - new Date(b.from).getTime()),
    [periods]
  );

  // Build x-axis labels from period start dates
  const periodLabels = useMemo(
    () =>
      sortedPeriods.map((p) => {
        const d = new Date(p.from);
        return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
      }),
    [sortedPeriods]
  );

  // Build chart data: one series per grade, filtered by selected step
  const chartData = useMemo(() => {
    return allGrades
      .map((grade, idx) => ({
        id: grade,
        color: GRADE_COLORS[idx % GRADE_COLORS.length],
        data: sortedPeriods
          .map((period, periodIdx) => {
            const entry = period.entries?.find((e) => e.grade === grade && e.step === selectedStep);
            if (!entry) return null;
            return {
              x: periodLabels[periodIdx],
              y: entry.monthly_amount / 100, // cents to EUR
            };
          })
          .filter((d): d is { x: string; y: number } => d !== null),
      }))
      .filter((series) => series.data.length > 0);
  }, [allGrades, sortedPeriods, selectedStep, periodLabels]);

  // Custom layer that renders % change labels between consecutive data points
  const PercentChangeLayer = useMemo(() => {
    return function PercentChangeLabels({
      series,
    }: {
      series: {
        id: string;
        color: string;
        data: { position: { x: number; y: number }; data: { y: number } }[];
      }[];
    }) {
      return (
        <g>
          {series.map((s) =>
            s.data.slice(1).map((point, i) => {
              const prev = s.data[i];
              const prevY = prev.data.y;
              const curY = point.data.y;
              if (prevY === 0) return null;
              const pct = ((curY - prevY) / prevY) * 100;
              const midX = (prev.position.x + point.position.x) / 2;
              const midY = (prev.position.y + point.position.y) / 2;
              return (
                <text
                  key={`${s.id}-${i}`}
                  x={midX}
                  y={midY - 8}
                  textAnchor="middle"
                  fontSize={10}
                  fontWeight={600}
                  fill={pct >= 0 ? '#22c55e' : '#ef4444'}
                >
                  {pct >= 0 ? '+' : ''}
                  {pct.toFixed(1)}%
                </text>
              );
            })
          )}
        </g>
      );
    };
  }, []);

  if (sortedPeriods.length < 2 || allGrades.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <h3 className="text-base font-medium">{t('payPlans.salaryChart')}</h3>
        <div className="flex items-center gap-2">
          <label className="text-muted-foreground text-sm">{t('payPlans.step')}</label>
          <Select value={String(selectedStep)} onValueChange={(v) => setSelectedStep(Number(v))}>
            <SelectTrigger className="w-20">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {allSteps.map((step) => (
                <SelectItem key={step} value={String(step)}>
                  {step}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <ExportableChart filename={`payplan-salary-step-${selectedStep}`} className="h-[350px]">
        <ResponsiveLine
          data={chartData}
          margin={{ top: 20, right: 120, bottom: 60, left: 80 }}
          xScale={{ type: 'point' }}
          yScale={{ type: 'linear', min: 'auto', max: 'auto' }}
          layers={[
            'grid',
            'markers',
            'axes',
            'areas',
            'crosshair',
            'lines',
            'points',
            PercentChangeLayer as any,
            'slices',
            'mesh',
            'legends',
          ]}
          curve="monotoneX"
          axisTop={null}
          axisRight={null}
          axisBottom={{
            tickSize: 5,
            tickPadding: 5,
            tickRotation: -30,
          }}
          axisLeft={{
            tickSize: 5,
            tickPadding: 5,
            format: (v) => formatCurrency(Number(v) * 100),
          }}
          colors={chartData.map((s) => s.color)}
          pointSize={8}
          pointColor={{ from: 'series.color' }}
          pointBorderWidth={2}
          pointBorderColor={{ theme: 'background' }}
          useMesh={true}
          enableSlices="x"
          sliceTooltip={({ slice }) => {
            // Find current x-index to compute % change from previous period
            const currentX = slice.points[0].data.xFormatted as string;
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
                <strong>{currentX}</strong>
                {slice.points.map((point) => {
                  const series = chartData.find((s) => s.id === point.seriesId);
                  const pointIdx = series?.data.findIndex((d) => d.x === currentX) ?? -1;
                  const prevValue = pointIdx > 0 ? series?.data[pointIdx - 1]?.y : undefined;
                  const currentValue = Number(point.data.yFormatted);
                  const pctChange =
                    prevValue != null && prevValue > 0
                      ? ((currentValue - prevValue) / prevValue) * 100
                      : undefined;
                  return (
                    <div
                      key={point.id}
                      style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}
                    >
                      <span
                        style={{
                          width: 10,
                          height: 10,
                          borderRadius: '50%',
                          background: point.seriesColor,
                          display: 'inline-block',
                        }}
                      />
                      {point.seriesId}: {formatCurrency(currentValue * 100)}
                      {pctChange != null && (
                        <span
                          style={{
                            color: pctChange >= 0 ? '#22c55e' : '#ef4444',
                            fontWeight: 600,
                            marginLeft: 4,
                          }}
                        >
                          {pctChange >= 0 ? '+' : ''}
                          {pctChange.toFixed(1)}%
                        </span>
                      )}
                    </div>
                  );
                })}
              </div>
            );
          }}
          legends={[
            {
              anchor: 'right',
              direction: 'column',
              justify: false,
              translateX: 110,
              translateY: 0,
              itemsSpacing: 2,
              itemDirection: 'left-to-right',
              itemWidth: 100,
              itemHeight: 18,
              itemOpacity: 0.85,
              symbolSize: 10,
              symbolShape: 'circle',
            },
          ]}
          theme={chartTheme}
        />
      </ExportableChart>
    </div>
  );
}
