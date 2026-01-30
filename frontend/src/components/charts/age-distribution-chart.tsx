'use client';

import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';
import type { AgeDistributionResponse } from '@/lib/api/types';

interface AgeDistributionChartProps {
  data: AgeDistributionResponse;
}

export function AgeDistributionChart({ data }: AgeDistributionChartProps) {
  const t = useTranslations();

  const chartData = data.distribution.map((bucket) => ({
    age: bucket.age_label.includes('+')
      ? t('statistics.ageSixPlus')
      : t('statistics.ageYears', { age: bucket.age_label }),
    [t('gender.male')]: bucket.male_count,
    [t('gender.female')]: bucket.female_count,
    [t('gender.diverse')]: bucket.diverse_count,
  }));

  const keys = [t('gender.male'), t('gender.female'), t('gender.diverse')];

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {t('statistics.totalChildren', { count: data.total_count })}
      </p>
      <div className="h-[300px]">
        <ResponsiveBar
          data={chartData}
          keys={keys}
          indexBy="age"
          margin={{ top: 20, right: 130, bottom: 50, left: 60 }}
          padding={0.3}
          groupMode="stacked"
          colors={['#3b82f6', '#ec4899', '#8b5cf6']}
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
          legends={[
            {
              dataFrom: 'keys',
              anchor: 'bottom-right',
              direction: 'column',
              justify: false,
              translateX: 120,
              translateY: 0,
              itemsSpacing: 2,
              itemWidth: 100,
              itemHeight: 20,
              itemDirection: 'left-to-right',
              itemOpacity: 0.85,
              symbolSize: 12,
              symbolShape: 'circle',
            },
          ]}
          role="application"
          ariaLabel={t('statistics.ageDistribution')}
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
