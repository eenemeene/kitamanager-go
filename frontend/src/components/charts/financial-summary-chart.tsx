'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';
import type { BarDatum, BarCustomLayerProps } from '@nivo/bar';
import { ExportableChart } from './exportable-chart';
import type { FinancialResponse } from '@/lib/api/types';
import { buildKitaYearBands, formatDateLabel, kitaYearLabel, chartTheme } from './chart-utils';
import { toLocalDateString } from '@/lib/utils/formatting';

interface FinancialSummaryChartProps {
  data: FinancialResponse;
}

type BandScale = ((v: string) => number | undefined) & { bandwidth(): number };

function formatEur(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

export function FinancialSummaryChart({ data }: FinancialSummaryChartProps) {
  const t = useTranslations();

  const balanceKey = t('statistics.cumulativeBalance');

  const rawDates = data.data_points.map((dp) => dp.date);
  const xLabels = rawDates.map(formatDateLabel);
  const kitaYearBands = useMemo(() => buildKitaYearBands(rawDates), [rawDates]);

  const { chartData, deltas } = useMemo(() => {
    let cumulative = 0;
    let currentKitaYear = '';
    const points: BarDatum[] = [];
    const monthlyDeltas: number[] = [];
    data.data_points.forEach((dp) => {
      const ky = kitaYearLabel(dp.date);
      if (ky !== currentKitaYear) {
        cumulative = 0;
        currentKitaYear = ky;
      }
      cumulative += dp.balance;
      monthlyDeltas.push(dp.balance);
      points.push({
        date: formatDateLabel(dp.date),
        [balanceKey]: cumulative / 100,
      });
    });
    return { chartData: points, deltas: monthlyDeltas };
  }, [data, balanceKey]);

  const todayStr = toLocalDateString(new Date());
  const todayLabel = formatDateLabel(todayStr);

  const KitaYearBackground = useMemo(() => {
    return function KitaYearBg({ xScale, innerHeight, innerWidth }: BarCustomLayerProps<BarDatum>) {
      const scale = xScale as unknown as BandScale;
      const bw = scale.bandwidth();

      return (
        <g>
          {kitaYearBands.map((band, i) => {
            const x0 = scale(xLabels[band.startIdx]) ?? 0;
            const x1 = (scale(xLabels[band.endIdx]) ?? 0) + bw;
            const clampedX0 = Math.max(0, x0);
            const clampedX1 = Math.min(innerWidth, x1);
            const width = clampedX1 - clampedX0;
            const midX = clampedX0 + width / 2;

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
                {i > 0 && (
                  <line
                    x1={clampedX0}
                    x2={clampedX0}
                    y1={0}
                    y2={innerHeight}
                    stroke="currentColor"
                    strokeWidth={1}
                    strokeDasharray="4 3"
                    opacity={0.2}
                  />
                )}
                {(() => {
                  const bracketY = innerHeight + 68;
                  const tickH = 4;
                  const labelY = bracketY + 14;
                  return (
                    <>
                      <line
                        x1={clampedX0 + 4}
                        x2={clampedX1 - 4}
                        y1={bracketY}
                        y2={bracketY}
                        stroke="currentColor"
                        strokeWidth={1}
                        opacity={0.3}
                      />
                      <line
                        x1={clampedX0 + 4}
                        x2={clampedX0 + 4}
                        y1={bracketY - tickH}
                        y2={bracketY}
                        stroke="currentColor"
                        strokeWidth={1}
                        opacity={0.3}
                      />
                      <line
                        x1={clampedX1 - 4}
                        x2={clampedX1 - 4}
                        y1={bracketY - tickH}
                        y2={bracketY}
                        stroke="currentColor"
                        strokeWidth={1}
                        opacity={0.3}
                      />
                      <line
                        x1={midX}
                        x2={midX}
                        y1={bracketY}
                        y2={bracketY + 4}
                        stroke="currentColor"
                        strokeWidth={1}
                        opacity={0.3}
                      />
                      <text
                        x={midX}
                        y={labelY}
                        textAnchor="middle"
                        fontSize={11}
                        fontWeight={500}
                        fill="currentColor"
                        opacity={0.5}
                      >
                        {t('statistics.kitaYear', { year: band.label })}
                      </text>
                    </>
                  );
                })()}
              </g>
            );
          })}
        </g>
      );
    };
  }, [kitaYearBands, xLabels, t]);

  const TodayMarker = useMemo(() => {
    return function TodayMarkerLayer({ xScale, innerHeight }: BarCustomLayerProps<BarDatum>) {
      const scale = xScale as unknown as BandScale;
      const x = scale(todayLabel);
      if (x === undefined) return null;
      const cx = x + scale.bandwidth() / 2;

      return (
        <g>
          <line
            x1={cx}
            x2={cx}
            y1={0}
            y2={innerHeight}
            stroke="hsl(var(--foreground))"
            strokeWidth={1}
            strokeDasharray="4 4"
          />
          <text x={cx} y={-4} textAnchor="middle" fontSize={11} fill="hsl(var(--muted-foreground))">
            {t('common.today')}
          </text>
        </g>
      );
    };
  }, [todayLabel, t]);

  return (
    <ExportableChart filename="financial-summary" className="h-[550px]">
      <ResponsiveBar
        data={chartData}
        keys={[balanceKey]}
        indexBy="date"
        margin={{ top: 40, right: 30, bottom: 130, left: 90 }}
        padding={0.3}
        valueScale={{ type: 'linear', min: 'auto' }}
        colors={({ data: d }) => ((d[balanceKey] as number) >= 0 ? '#22c55e' : '#ef4444')}
        layers={[
          KitaYearBackground,
          'grid',
          'axes',
          'bars',
          TodayMarker,
          'markers',
          'legends',
          'annotations',
        ]}
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
          format: (v) =>
            Number(v).toLocaleString('de-DE', {
              style: 'currency',
              currency: 'EUR',
              maximumFractionDigits: 0,
            }),
        }}
        enableLabel={true}
        label={(d) => formatEur((d.value as number) * 100)}
        labelSkipWidth={50}
        labelSkipHeight={16}
        labelTextColor={{ from: 'color', modifiers: [['darker', 2]] }}
        tooltip={({ indexValue, value, color }) => {
          const idx = chartData.findIndex((d) => d.date === indexValue);
          const delta = idx >= 0 ? deltas[idx] : 0;
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
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
                <span
                  style={{
                    width: 10,
                    height: 10,
                    borderRadius: '50%',
                    background: color,
                    display: 'inline-block',
                  }}
                />
                {formatEur((value as number) * 100)}
              </div>
              {delta !== 0 && (
                <div
                  style={{
                    marginTop: 4,
                    color: delta > 0 ? '#22c55e' : '#ef4444',
                    fontSize: 12,
                  }}
                >
                  {delta > 0 ? '+' : ''}
                  {formatEur(delta)} {t('statistics.monthlyChange')}
                </div>
              )}
            </div>
          );
        }}
        legends={[
          {
            dataFrom: 'keys',
            anchor: 'top',
            direction: 'row',
            justify: false,
            translateX: 0,
            translateY: -35,
            itemsSpacing: 4,
            itemDirection: 'left-to-right',
            itemWidth: 200,
            itemHeight: 20,
            itemOpacity: 0.85,
            symbolSize: 12,
            symbolShape: 'circle',
          },
        ]}
        role="application"
        ariaLabel={t('statistics.financialSummary')}
        theme={chartTheme}
      />
    </ExportableChart>
  );
}
