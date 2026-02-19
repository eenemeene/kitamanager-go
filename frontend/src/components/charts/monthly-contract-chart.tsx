'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveLine } from '@nivo/line';
import type { StaffingHoursResponse } from '@/lib/api/types';
import {
  buildKitaYearBands,
  formatDateLabel,
  createKitaYearBackgroundLayer,
  createTodayMarker,
  chartTheme,
} from './chart-utils';

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

  const todayStr = new Date().toISOString().slice(0, 10);
  const todayLabel = formatDateLabel(todayStr);

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

  return (
    <div className="h-[350px]">
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
        pointSize={8}
        pointColor={{ theme: 'background' }}
        pointBorderWidth={2}
        pointBorderColor={{ from: 'serieColor' }}
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
    </div>
  );
}
