'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveLine, type CustomLayerProps } from '@nivo/line';
import type { StaffingHoursResponse } from '@/lib/api/types';

interface StaffingHoursChartProps {
  data: StaffingHoursResponse;
}

/** Returns the Kita year label for a given date (Aug–Jul). e.g. August 2024 → "24/25" */
function kitaYearLabel(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  const month = date.getMonth(); // 0-indexed
  const year = date.getFullYear();
  const startYear = month >= 7 ? year : year - 1; // Aug (7) starts a new Kita year
  const sy = String(startYear).slice(2);
  const ey = String(startYear + 1).slice(2);
  return `${sy}/${ey}`;
}

interface KitaYearBand {
  label: string;
  startIdx: number;
  endIdx: number;
}

/** Groups consecutive data point indices by their Kita year */
function buildKitaYearBands(dates: string[]): KitaYearBand[] {
  if (dates.length === 0) return [];
  const bands: KitaYearBand[] = [];
  let currentLabel = kitaYearLabel(dates[0]);
  let startIdx = 0;
  for (let i = 1; i < dates.length; i++) {
    const label = kitaYearLabel(dates[i]);
    if (label !== currentLabel) {
      bands.push({ label: currentLabel, startIdx, endIdx: i - 1 });
      currentLabel = label;
      startIdx = i;
    }
  }
  bands.push({ label: currentLabel, startIdx, endIdx: dates.length - 1 });
  return bands;
}

export function StaffingHoursChart({ data }: StaffingHoursChartProps) {
  const t = useTranslations();

  // Format dates as "Jan 25", "Feb 25", etc.
  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr + 'T00:00:00');
    return date.toLocaleDateString('en-US', { month: 'short', year: '2-digit' });
  };

  const rawDates = data.data_points.map((dp) => dp.date);
  const xLabels = rawDates.map(formatDate);
  const kitaYearBands = useMemo(() => buildKitaYearBands(rawDates), [rawDates]);

  // Custom Nivo layer that draws alternating background bands per Kita year
  const KitaYearBackgroundLayer = useMemo(() => {
    return function KitaYearBg({ xScale, innerHeight, innerWidth }: CustomLayerProps) {
      const scale = xScale as unknown as (value: string) => number;
      const step = xLabels.length > 1 ? scale(xLabels[1]) - scale(xLabels[0]) : innerWidth;

      return (
        <g>
          {kitaYearBands.map((band, i) => {
            const x0 = scale(xLabels[band.startIdx]) - step / 2;
            const x1 = scale(xLabels[band.endIdx]) + step / 2;
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

  // Find today marker position
  const today = new Date();
  const todayStr = today.toISOString().slice(0, 10);

  const chartData = [
    {
      id: t('statistics.requiredHours'),
      color: '#f59e0b',
      data: data.data_points.map((dp) => ({
        x: formatDate(dp.date),
        y: Math.round(dp.required_hours * 100) / 100,
      })),
    },
    {
      id: t('statistics.availableHours'),
      color: '#3b82f6',
      data: data.data_points.map((dp) => ({
        x: formatDate(dp.date),
        y: dp.available_hours,
      })),
    },
  ];

  // Find the x label for today's month
  const todayLabel = formatDate(todayStr);

  return (
    <div className="h-[300px]">
      <ResponsiveLine
        data={chartData}
        margin={{ top: 20, right: 110, bottom: 50, left: 60 }}
        xScale={{ type: 'point' }}
        yScale={{ type: 'linear', min: 'auto', max: 'auto', stacked: false }}
        layers={[
          KitaYearBackgroundLayer,
          'grid',
          'markers',
          'axes',
          'areas',
          'crosshair',
          'lines',
          'points',
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
          tickRotation: -45,
        }}
        axisLeft={{
          tickSize: 5,
          tickPadding: 5,
          tickRotation: 0,
        }}
        colors={['#f59e0b', '#3b82f6']}
        pointSize={8}
        pointColor={{ theme: 'background' }}
        pointBorderWidth={2}
        pointBorderColor={{ from: 'serieColor' }}
        pointLabelYOffset={-12}
        useMesh={true}
        enableSlices="x"
        markers={[
          {
            axis: 'x',
            value: todayLabel,
            lineStyle: {
              stroke: 'hsl(var(--foreground))',
              strokeWidth: 1,
              strokeDasharray: '4 4',
            },
            legend: t('common.today'),
            legendPosition: 'top',
            textStyle: {
              fill: 'hsl(var(--muted-foreground))',
              fontSize: 11,
            },
          },
        ]}
        legends={[
          {
            anchor: 'bottom-right',
            direction: 'column',
            justify: false,
            translateX: 100,
            translateY: 0,
            itemsSpacing: 0,
            itemDirection: 'left-to-right',
            itemWidth: 80,
            itemHeight: 20,
            itemOpacity: 0.85,
            symbolSize: 12,
            symbolShape: 'circle',
          },
        ]}
        theme={{
          axis: {
            ticks: {
              text: {
                fill: 'hsl(var(--muted-foreground))',
              },
            },
          },
          grid: {
            line: {
              stroke: 'hsl(var(--border))',
            },
          },
          legends: {
            text: {
              fill: 'hsl(var(--foreground))',
            },
          },
          crosshair: {
            line: {
              stroke: 'hsl(var(--foreground))',
              strokeWidth: 1,
              strokeOpacity: 0.35,
            },
          },
          tooltip: {
            container: {
              background: 'hsl(var(--background))',
              color: 'hsl(var(--foreground))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '6px',
            },
          },
        }}
      />
    </div>
  );
}
