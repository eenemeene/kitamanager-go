'use client';

import { useTranslations } from 'next-intl';
import { ResponsiveLine } from '@nivo/line';
import type { ChildrenContractCountByMonthResponse } from '@/lib/api/types';

interface MonthlyContractChartProps {
  data: ChildrenContractCountByMonthResponse;
}

const COLORS = ['#3b82f6', '#22c55e', '#f59e0b', '#ef4444', '#8b5cf6'];

export function MonthlyContractChart({ data }: MonthlyContractChartProps) {
  const t = useTranslations();

  const monthKeys = [
    'jan',
    'feb',
    'mar',
    'apr',
    'may',
    'jun',
    'jul',
    'aug',
    'sep',
    'oct',
    'nov',
    'dec',
  ];

  // Transform data for Nivo line chart
  const chartData = data.years.map((year, index) => ({
    id: year.year.toString(),
    color: COLORS[index % COLORS.length],
    data: monthKeys.map((month, monthIndex) => ({
      x: t(`months.${month}`),
      y: year.counts[monthIndex] || 0,
    })),
  }));

  return (
    <div className="h-[300px]">
      <ResponsiveLine
        data={chartData}
        margin={{ top: 20, right: 110, bottom: 50, left: 60 }}
        xScale={{ type: 'point' }}
        yScale={{ type: 'linear', min: 'auto', max: 'auto', stacked: false }}
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
        colors={COLORS}
        pointSize={8}
        pointColor={{ theme: 'background' }}
        pointBorderWidth={2}
        pointBorderColor={{ from: 'serieColor' }}
        pointLabelYOffset={-12}
        useMesh={true}
        enableSlices="x"
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
