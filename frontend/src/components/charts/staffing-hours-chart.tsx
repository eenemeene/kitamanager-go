'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveLine, type CustomLayerProps } from '@nivo/line';
import { scaleLinear } from 'd3-scale';
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

  // Compute balance percentages for the bar layer
  const balancePercentages = useMemo(
    () =>
      data.data_points.map((dp) =>
        dp.required_hours > 0
          ? Math.round(((dp.available_hours - dp.required_hours) / dp.required_hours) * 1000) / 10
          : 0
      ),
    [data.data_points]
  );

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

  // Custom layer that draws balance percentage bars behind the lines
  const BalanceBarsLayer = useMemo(() => {
    return function BalanceBars({ xScale, innerHeight, innerWidth }: CustomLayerProps) {
      const scale = xScale as unknown as (value: string) => number;
      const step = xLabels.length > 1 ? scale(xLabels[1]) - scale(xLabels[0]) : innerWidth;
      const barWidth = step * 0.5;

      // Build a symmetric y-scale for percentages
      const maxAbs = Math.max(10, ...balancePercentages.map(Math.abs));
      const pctScale = scaleLinear().domain([-maxAbs, maxAbs]).range([innerHeight, 0]);
      const zeroY = pctScale(0);

      // Right-axis ticks
      const ticks = pctScale.ticks(5);

      return (
        <g>
          {/* Bars */}
          {xLabels.map((label, i) => {
            const pct = balancePercentages[i];
            const cx = scale(label);
            const barY = pct >= 0 ? pctScale(pct) : zeroY;
            const barH = Math.abs(pctScale(pct) - zeroY);

            return (
              <rect
                key={label}
                x={cx - barWidth / 2}
                y={barY}
                width={barWidth}
                height={barH}
                fill={pct >= 0 ? '#22c55e' : '#ef4444'}
                opacity={0.2}
                rx={2}
              />
            );
          })}
          {/* Zero line */}
          <line
            x1={0}
            x2={innerWidth}
            y1={zeroY}
            y2={zeroY}
            stroke="hsl(var(--muted-foreground))"
            strokeWidth={1}
            strokeDasharray="3 3"
            opacity={0.5}
          />
          {/* Right axis ticks */}
          {ticks.map((tick) => (
            <g key={tick} transform={`translate(${innerWidth}, ${pctScale(tick)})`}>
              <line x1={0} x2={5} y1={0} y2={0} stroke="hsl(var(--muted-foreground))" />
              <text
                x={8}
                y={0}
                dominantBaseline="central"
                fontSize={10}
                fill="hsl(var(--muted-foreground))"
              >
                {tick > 0 ? '+' : ''}
                {tick}%
              </text>
            </g>
          ))}
        </g>
      );
    };
  }, [xLabels, balancePercentages]);

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
    <div className="h-[350px]">
      <ResponsiveLine
        data={chartData}
        margin={{ top: 20, right: 60, bottom: 50, left: 60 }}
        xScale={{ type: 'point' }}
        yScale={{ type: 'linear', min: 'auto', max: 'auto', stacked: false }}
        layers={[
          KitaYearBackgroundLayer,
          BalanceBarsLayer,
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
        sliceTooltip={({ slice }) => {
          const idx = xLabels.indexOf(slice.points[0].data.xFormatted as string);
          const pct = idx >= 0 ? balancePercentages[idx] : null;
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
              <strong>{slice.points[0].data.xFormatted}</strong>
              {slice.points.map((point) => (
                <div
                  key={point.id}
                  style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}
                >
                  <span
                    style={{
                      width: 10,
                      height: 10,
                      borderRadius: '50%',
                      background: point.serieColor,
                      display: 'inline-block',
                    }}
                  />
                  {point.serieId}: {point.data.yFormatted}h
                </div>
              ))}
              {pct !== null && (
                <div
                  style={{ marginTop: 6, paddingTop: 6, borderTop: '1px solid hsl(var(--border))' }}
                >
                  <span
                    style={{
                      color: pct >= 0 ? '#22c55e' : '#ef4444',
                      fontWeight: 600,
                    }}
                  >
                    {t('statistics.balancePercentage')}: {pct > 0 ? '+' : ''}
                    {pct}%
                  </span>
                </div>
              )}
            </div>
          );
        }}
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
            anchor: 'top-left',
            direction: 'row',
            justify: false,
            translateX: 0,
            translateY: -20,
            itemsSpacing: 16,
            itemDirection: 'left-to-right',
            itemWidth: 120,
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
