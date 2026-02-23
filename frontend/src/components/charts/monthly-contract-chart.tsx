'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveLine } from '@nivo/line';
import { ExportableChart } from './exportable-chart';
import type { StaffingHoursResponse } from '@/lib/api/types';
import {
  buildKitaYearBands,
  formatDateLabel,
  createKitaYearBackgroundLayer,
  createTodayMarker,
  chartTheme,
} from './chart-utils';
import { toLocalDateString } from '@/lib/utils/formatting';

interface MonthlyContractChartProps {
  data: StaffingHoursResponse;
}

export function MonthlyContractChart({ data }: MonthlyContractChartProps) {
  const t = useTranslations();

  const rawDates = data.data_points.map((dp) => dp.date);
  const xLabels = rawDates.map(formatDateLabel);
  const kitaYearBands = useMemo(() => buildKitaYearBands(rawDates), [rawDates]);

  const KitaYearBackgroundLayer = useMemo(
    () =>
      createKitaYearBackgroundLayer(kitaYearBands, xLabels, (label) =>
        t('statistics.kitaYear', { year: label })
      ),
    [kitaYearBands, xLabels, t]
  );

  const todayStr = toLocalDateString(new Date());
  const todayLabel = formatDateLabel(todayStr);

  const counts = data.data_points.map((dp) => dp.child_count);

  const chartData = [
    {
      id: t('statistics.childrenContractCount'),
      color: '#3b82f6',
      data: data.data_points.map((dp) => ({
        x: formatDateLabel(dp.date),
        y: dp.child_count,
      })),
    },
  ];

  const TrendArrows = useMemo(() => {
    return function TrendArrowsLayer({
      xScale,
      yScale,
    }: {
      xScale: (v: string) => number;
      yScale: (v: number) => number;
    }) {
      return (
        <g>
          {xLabels.map((label, i) => {
            if (i === 0) return null;
            const diff = counts[i] - counts[i - 1];
            if (diff === 0) return null;

            const x0 = xScale(xLabels[i - 1]);
            const x1 = xScale(label);
            const y0 = yScale(counts[i - 1]);
            const y1 = yScale(counts[i]);
            const midX = (x0 + x1) / 2;
            const midY = (y0 + y1) / 2;

            const isUp = diff > 0;
            const color = isUp ? '#16a34a' : '#dc2626';
            const arrow = isUp ? '▲' : '▼';
            const offsetY = isUp ? 14 : -14;

            return (
              <g key={i}>
                <text
                  x={midX}
                  y={midY + offsetY - 6}
                  textAnchor="middle"
                  fontSize={9}
                  fill={color}
                  fontWeight={600}
                >
                  {arrow}
                </text>
                <text
                  x={midX}
                  y={midY + offsetY + 8}
                  textAnchor="middle"
                  fontSize={10}
                  fill={color}
                  fontWeight={600}
                >
                  {isUp ? '+' : ''}
                  {diff}
                </text>
              </g>
            );
          })}
        </g>
      );
    };
  }, [xLabels, counts]);

  return (
    <ExportableChart filename="children-contracts" className="h-[350px]">
      <ResponsiveLine
        data={chartData}
        margin={{ top: 20, right: 30, bottom: 80, left: 60 }}
        xScale={{ type: 'point' }}
        yScale={{ type: 'linear', min: 'auto', max: 'auto', stacked: false }}
        layers={[
          KitaYearBackgroundLayer as any,
          'grid',
          'markers',
          'axes',
          'areas',
          'crosshair',
          'lines',
          TrendArrows as any,
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
        colors={['#3b82f6']}
        pointSize={6}
        pointColor={{ from: 'series.color' }}
        pointBorderWidth={2}
        pointBorderColor={{ theme: 'background' }}
        pointLabelYOffset={-12}
        useMesh={true}
        enableSlices="x"
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
            itemWidth: 200,
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
