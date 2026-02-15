'use client';

import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';

export interface SectionStaffingData {
  sectionName: string;
  required: number;
  available: number;
}

interface SectionStaffingChartProps {
  data: SectionStaffingData[];
}

export function SectionStaffingChart({ data }: SectionStaffingChartProps) {
  const t = useTranslations();

  const requiredKey = t('statistics.requiredHours');
  const availableKey = t('statistics.availableHours');

  const chartData = data.map((d) => ({
    section: d.sectionName,
    [requiredKey]: Math.round(d.required * 100) / 100,
    [availableKey]: Math.round(d.available * 100) / 100,
  }));

  const keys = [requiredKey, availableKey];

  return (
    <div className="h-[300px]">
      <ResponsiveBar
        data={chartData}
        keys={keys}
        indexBy="section"
        margin={{ top: 20, right: 130, bottom: 50, left: 60 }}
        padding={0.3}
        groupMode="grouped"
        colors={['#f59e0b', '#3b82f6']}
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
        tooltip={({ id, value, indexValue }) => {
          const entry = data.find((d) => d.sectionName === indexValue);
          const balance =
            entry && entry.required > 0
              ? Math.round(((entry.available - entry.required) / entry.required) * 1000) / 10
              : null;
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
                {id}: {value}h
              </div>
              {balance !== null && (
                <div
                  style={{ marginTop: 6, paddingTop: 6, borderTop: '1px solid hsl(var(--border))' }}
                >
                  <span
                    style={{
                      color: balance >= 0 ? '#22c55e' : '#ef4444',
                      fontWeight: 600,
                    }}
                  >
                    {t('statistics.balancePercentage')}: {balance > 0 ? '+' : ''}
                    {balance}%
                  </span>
                </div>
              )}
            </div>
          );
        }}
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
        ariaLabel={t('statistics.sectionStaffing')}
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
  );
}
