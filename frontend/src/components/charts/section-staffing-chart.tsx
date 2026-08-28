'use client';

import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';
import { chartTheme } from './chart-utils';
import { ExportableChart } from './exportable-chart';

export interface SectionStaffingData {
  sectionName: string;
  required: number;
  available: number;
}

interface SectionStaffingChartProps {
  data: SectionStaffingData[];
}

export interface SectionStaffingChartRow {
  section: string;
  available: number;
  required: number;
  // The percentage value lives under a translated key (Nivo's
  // `keys` mechanism) so the legend localises automatically. Tests
  // pass an explicit balanceKey to keep them locale-independent.
  [balanceKey: string]: string | number;
}

/** Compute the symmetric domain max (with ~10% padding, floor at 10%). */
export function computeSymmetricDomainMax(percentages: number[]): number {
  const rawMax = Math.max(10, ...percentages.map(Math.abs));
  return Math.ceil(rawMax * 1.1);
}

/**
 * Build per-section chart rows. Pure function exported so the per-section
 * percentage math and rounding rules are unit-tested without rendering.
 *
 * Edge cases handled:
 *  - `required === 0` → percentage is 0 (avoids Infinity)
 *  - rounding hours to whole numbers for the y-axis tick formatter
 *  - rounding percentages to 0.1% so labels read +12.3% / -7.5%
 */
export function buildSectionStaffingRows(
  data: SectionStaffingData[],
  balanceKey: string
): SectionStaffingChartRow[] {
  return data.map((d) => {
    const pct =
      d.required > 0 ? Math.round(((d.available - d.required) / d.required) * 1000) / 10 : 0;
    return {
      section: d.sectionName,
      [balanceKey]: pct,
      available: Math.round(d.available),
      required: Math.round(d.required),
    };
  });
}

export function SectionStaffingChart({ data }: SectionStaffingChartProps) {
  const t = useTranslations();

  const balanceKey = t('statistics.balancePercentage');

  const chartData = buildSectionStaffingRows(data, balanceKey);
  const maxAbs = computeSymmetricDomainMax(chartData.map((d) => d[balanceKey] as number));

  return (
    <div className="space-y-2">
      <p className="text-muted-foreground text-sm">{t('statistics.sectionStaffingDescription')}</p>
      <ExportableChart filename="section-staffing" className="h-[300px]">
        <ResponsiveBar
          data={chartData}
          keys={[balanceKey]}
          indexBy="section"
          layout="horizontal"
          margin={{ top: 10, right: 40, bottom: 60, left: 220 }}
          padding={0.3}
          valueScale={{ type: 'linear', min: -maxAbs, max: maxAbs }}
          colors={({ value }) => ((value ?? 0) >= 0 ? '#22c55e' : '#ef4444')}
          borderRadius={3}
          enableGridX={true}
          gridXValues={Array.from(
            { length: Math.floor(maxAbs / 10) * 2 + 1 },
            (_, i) => -Math.floor(maxAbs / 10) * 10 + i * 10
          )}
          enableGridY={true}
          enableLabel={true}
          label={({ value }) => `${(value ?? 0) > 0 ? '+' : ''}${value}%`}
          labelSkipWidth={20}
          labelTextColor="#fff"
          axisTop={null}
          axisRight={null}
          axisBottom={{
            tickSize: 5,
            tickPadding: 5,
            tickValues: Array.from(
              { length: Math.floor(maxAbs / 10) * 2 + 1 },
              (_, i) => -Math.floor(maxAbs / 10) * 10 + i * 10
            ),
            format: (v) => `${v}%`,
          }}
          axisLeft={{
            tickSize: 0,
            tickPadding: 8,
            format: (v) => {
              const entry = chartData.find((d) => d.section === v);
              if (!entry) return String(v);
              return `${v}  (${entry.available}h / ${entry.required}h)`;
            },
          }}
          tooltip={({ indexValue, value }) => {
            const entry = data.find((d) => d.sectionName === indexValue);
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
                <div style={{ marginTop: 4 }}>
                  <span
                    style={{
                      color: (value ?? 0) >= 0 ? '#22c55e' : '#ef4444',
                      fontWeight: 600,
                    }}
                  >
                    {(value ?? 0) > 0 ? '+' : ''}
                    {value}%
                  </span>
                </div>
                {entry && (
                  <div
                    style={{
                      marginTop: 6,
                      paddingTop: 6,
                      borderTop: '1px solid hsl(var(--border))',
                      fontSize: 12,
                      opacity: 0.8,
                    }}
                  >
                    <div>
                      {t('statistics.availableHours')}: {Math.round(entry.available)}h
                    </div>
                    <div>
                      {t('statistics.requiredHours')}: {Math.round(entry.required)}h
                    </div>
                  </div>
                )}
              </div>
            );
          }}
          markers={[
            {
              axis: 'x',
              value: 0,
              lineStyle: {
                stroke: 'hsl(var(--muted-foreground))',
                strokeWidth: 1,
                strokeDasharray: '3 3',
              },
            },
          ]}
          legends={[
            {
              dataFrom: 'keys',
              anchor: 'bottom',
              direction: 'row',
              justify: false,
              translateX: 0,
              translateY: 50,
              itemsSpacing: 4,
              itemDirection: 'left-to-right',
              itemWidth: 100,
              itemHeight: 20,
              itemOpacity: 0.85,
              symbolSize: 12,
              symbolShape: 'circle',
            },
          ]}
          role="img"
          ariaLabel={t('statistics.sectionStaffing')}
          theme={chartTheme}
        />
      </ExportableChart>
    </div>
  );
}
