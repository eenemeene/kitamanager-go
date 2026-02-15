'use client';

import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';
import type { ContractPropertiesDistributionResponse } from '@/lib/api/types';

interface ContractPropertiesChartProps {
  data: ContractPropertiesDistributionResponse;
}

export function ContractPropertiesChart({ data }: ContractPropertiesChartProps) {
  const t = useTranslations();

  const chartData = data.properties
    .map((p) => ({
      id: `${p.key}: ${p.value}`,
      label: `${p.key}: ${p.value}`,
      value: p.count,
    }))
    .reverse(); // reverse for horizontal so top-most is first alphabetically

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {t('statistics.totalChildren', { count: data.total_children })}
      </p>
      <div className="h-[300px]">
        <ResponsiveBar
          data={chartData}
          keys={['value']}
          indexBy="id"
          layout="horizontal"
          margin={{ top: 10, right: 20, bottom: 50, left: 180 }}
          padding={0.3}
          colors={['#3b82f6']}
          borderColor={{ from: 'color', modifiers: [['darker', 1.6]] }}
          axisTop={null}
          axisRight={null}
          axisBottom={{
            tickSize: 5,
            tickPadding: 5,
            tickRotation: 0,
          }}
          axisLeft={{
            tickSize: 5,
            tickPadding: 5,
            tickRotation: 0,
          }}
          enableLabel={true}
          labelSkipWidth={12}
          labelSkipHeight={12}
          labelTextColor={{ from: 'color', modifiers: [['brighter', 3]] }}
          role="application"
          ariaLabel={t('statistics.contractProperties')}
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
    </div>
  );
}
