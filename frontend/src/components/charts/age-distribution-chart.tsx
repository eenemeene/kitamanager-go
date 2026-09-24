'use client';

import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';
import { ExportableChart } from './exportable-chart';
import type { AgeDistributionResponse } from '@/lib/api/types';

interface AgeDistributionChartProps {
  data: AgeDistributionResponse;
}

export function AgeDistributionChart({ data }: AgeDistributionChartProps) {
  const t = useTranslations();

  // Children whose gender is none of the three were counted in the total but
  // drawn in no segment, so the bars added up to less than the caption above
  // them. The segment only appears when there are such children, which for
  // clean data is never.
  const hasUnknown = data.distribution.some((b) => (b.unknown_count ?? 0) > 0);

  const chartData = data.distribution.map((bucket) => {
    const ageLabel = bucket.age_label ?? '';
    const row: Record<string, string | number> = {
      age: ageLabel.includes('+')
        ? t('statistics.ageSixPlus')
        : t('statistics.ageYears', { age: ageLabel }),
      [t('gender.male')]: bucket.male_count ?? 0,
      [t('gender.female')]: bucket.female_count ?? 0,
      [t('gender.diverse')]: bucket.diverse_count ?? 0,
    };
    if (hasUnknown) {
      row[t('gender.unknown')] = bucket.unknown_count ?? 0;
    }
    return row;
  });

  const keys = hasUnknown
    ? [t('gender.male'), t('gender.female'), t('gender.diverse'), t('gender.unknown')]
    : [t('gender.male'), t('gender.female'), t('gender.diverse')];

  return (
    <div className="space-y-4">
      <p className="text-muted-foreground text-sm">
        {t('statistics.totalChildren', { count: data.total_count })}
      </p>
      <ExportableChart filename="age-distribution" className="h-[300px]">
        <ResponsiveBar
          data={chartData}
          keys={keys}
          indexBy="age"
          margin={{ top: 20, right: 130, bottom: 50, left: 60 }}
          padding={0.3}
          groupMode="stacked"
          colors={['#3b82f6', '#ec4899', '#8b5cf6', '#94a3b8']}
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
            legend: t('statistics.axisChildren'),
            legendPosition: 'middle',
            legendOffset: -50,
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
          role="img"
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
      </ExportableChart>
    </div>
  );
}
