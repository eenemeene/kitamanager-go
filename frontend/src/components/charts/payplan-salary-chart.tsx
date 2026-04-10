'use client';

import { useMemo, useState } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';
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

/** Parse grade string into [number, suffix] for natural sorting (e.g. "S8a" → [8, "a"]) */
function parseGrade(g: string): [number, string] {
  const match = g.match(/^[A-Za-z]*(\d+)(.*)$/);
  return match ? [parseInt(match[1]), match[2]] : [0, g];
}

/** A distinct color palette for up to 15 grade bars. */
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
      allGrades: Array.from(gradeSet).sort((a, b) => {
        const [numA, suffA] = parseGrade(a);
        const [numB, suffB] = parseGrade(b);
        if (numA !== numB) return numA - numB;
        return suffA.localeCompare(suffB);
      }),
    };
  }, [periods]);

  const [selectedStep, setSelectedStep] = useState<number>(() => allSteps[0] ?? 1);

  // Sort periods chronologically
  const sortedPeriods = useMemo(
    () => [...periods].sort((a, b) => new Date(a.from).getTime() - new Date(b.from).getTime()),
    [periods]
  );

  // Build bar chart data: one object per period, with a key per grade
  const { barData, gradeColorMap } = useMemo(() => {
    const colorMap: Record<string, string> = {};
    allGrades.forEach((grade, idx) => {
      colorMap[grade] = GRADE_COLORS[idx % GRADE_COLORS.length];
    });

    const data = sortedPeriods.map((period) => {
      const d = new Date(period.from);
      const label = d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
      const row: Record<string, string | number> = { period: label };
      for (const grade of allGrades) {
        const entry = period.entries?.find((e) => e.grade === grade && e.step === selectedStep);
        if (entry) {
          row[grade] = entry.monthly_amount / 100; // cents to EUR
        }
      }
      return row;
    });

    return { barData: data, gradeColorMap: colorMap };
  }, [allGrades, sortedPeriods, selectedStep]);

  // Compute which grades actually have data for the selected step
  const activeGrades = useMemo(
    () => allGrades.filter((grade) => barData.some((row) => row[grade] !== undefined)),
    [allGrades, barData]
  );

  if (sortedPeriods.length < 2 || activeGrades.length === 0) {
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
      <ExportableChart filename={`payplan-salary-step-${selectedStep}`} className="h-[600px]">
        <ResponsiveBar
          data={barData}
          keys={activeGrades}
          indexBy="period"
          groupMode="grouped"
          margin={{ top: 20, right: 130, bottom: 60, left: 80 }}
          padding={0.15}
          innerPadding={1}
          enableLabel={false}
          valueScale={{ type: 'linear' }}
          indexScale={{ type: 'band', round: true }}
          colors={(bar) => gradeColorMap[bar.id as string] ?? '#888'}
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
          tooltip={({ id, value, indexValue }) => {
            // Find previous period value for % change
            const periodIdx = barData.findIndex((row) => row.period === indexValue);
            const prevValue =
              periodIdx > 0 ? (barData[periodIdx - 1][id] as number | undefined) : undefined;
            const pctChange =
              prevValue != null && prevValue > 0
                ? ((value - prevValue) / prevValue) * 100
                : undefined;
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
                <strong>{indexValue}</strong>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
                  <span
                    style={{
                      width: 10,
                      height: 10,
                      borderRadius: '50%',
                      background: gradeColorMap[id as string],
                      display: 'inline-block',
                    }}
                  />
                  {id}: {formatCurrency(value * 100)}
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
              </div>
            );
          }}
          legends={[
            {
              dataFrom: 'keys',
              anchor: 'right',
              direction: 'column',
              justify: false,
              translateX: 120,
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
