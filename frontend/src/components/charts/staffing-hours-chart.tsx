'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveLine } from '@nivo/line';
import { ExportableChart } from './exportable-chart';
import { scaleLinear } from 'd3-scale';
import type { StaffingHoursResponse } from '@/lib/api/types';
import {
  buildKitaYearBands,
  formatDateLabel,
  createKitaYearBackgroundLayer,
  createTodayMarker,
  chartTheme,
} from './chart-utils';
import { toLocalDateString } from '@/lib/utils/formatting';

interface StaffingHoursChartProps {
  data: StaffingHoursResponse;
}

export function StaffingHoursChart({ data }: StaffingHoursChartProps) {
  const t = useTranslations();

  const rawDates = data.data_points.map((dp) => dp.date);
  const xLabels = rawDates.map(formatDateLabel);
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
  const KitaYearBackgroundLayer = useMemo(
    () =>
      createKitaYearBackgroundLayer(kitaYearBands, xLabels, (label) =>
        t('statistics.kitaYear', { year: label })
      ),
    [kitaYearBands, xLabels, t]
  );

  // Custom layer that draws balance percentage bars behind the lines
  const BalanceBarsLayer = useMemo(() => {
    return function BalanceBars({
      xScale,
      innerHeight,
      innerWidth,
    }: {
      xScale: (value: string) => number;
      innerHeight: number;
      innerWidth: number;
    }) {
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
          {/* Bars with percentage labels */}
          {xLabels.map((label, i) => {
            const pct = balancePercentages[i];
            const cx = scale(label);
            const barY = pct >= 0 ? pctScale(pct) : zeroY;
            const barH = Math.abs(pctScale(pct) - zeroY);
            const color = pct >= 0 ? '#22c55e' : '#ef4444';
            // Label position: above bar for positive, below bar for negative
            const labelY = pct >= 0 ? barY - 3 : barY + barH + 10;

            return (
              <g key={label}>
                <rect
                  x={cx - barWidth / 2}
                  y={barY}
                  width={barWidth}
                  height={barH}
                  fill={color}
                  opacity={0.2}
                  rx={2}
                />
                {pct !== 0 && (
                  <text
                    x={cx}
                    y={labelY}
                    textAnchor="middle"
                    fontSize={9}
                    fontWeight={600}
                    fill={color}
                  >
                    {pct > 0 ? '+' : ''}
                    {pct}%
                  </text>
                )}
              </g>
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
  const todayStr = toLocalDateString(new Date());

  const chartData = [
    {
      id: t('statistics.requiredHours'),
      color: '#f59e0b',
      data: data.data_points.map((dp) => ({
        x: formatDateLabel(dp.date),
        y: Math.round(dp.required_hours * 100) / 100,
      })),
    },
    {
      id: t('statistics.availableHours'),
      color: '#3b82f6',
      data: data.data_points.map((dp) => ({
        x: formatDateLabel(dp.date),
        y: dp.available_hours,
      })),
    },
  ];

  // Find the x label for today's month
  const todayLabel = formatDateLabel(todayStr);

  return (
    <ExportableChart filename="staffing-hours" className="h-[350px]">
      <ResponsiveLine
        data={chartData}
        margin={{ top: 20, right: 60, bottom: 80, left: 60 }}
        xScale={{ type: 'point' }}
        yScale={{ type: 'linear', min: 'auto', max: 'auto', stacked: false }}
        layers={[
          KitaYearBackgroundLayer as any,

          BalanceBarsLayer as any,
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
        pointSize={6}
        pointColor={{ from: 'series.color' }}
        pointBorderWidth={2}
        pointBorderColor={{ theme: 'background' }}
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
                      background: point.seriesColor,
                      display: 'inline-block',
                    }}
                  />
                  {point.seriesId}: {point.data.yFormatted}h
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
        markers={[createTodayMarker(todayLabel, t('common.today'))]}
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
        theme={chartTheme}
      />
    </ExportableChart>
  );
}
