'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveLine } from '@nivo/line';
import type { FinancialResponse } from '@/lib/api/types';
import {
  buildKitaYearBands,
  formatDateLabel,
  createKitaYearBackgroundLayer,
  createTodayMarker,
  chartTheme,
} from './chart-utils';

interface FinancialsChartProps {
  data: FinancialResponse;
}

/** Convert cents to EUR with 2 decimal places */
function centsToEur(cents: number): number {
  return Math.round(cents) / 100;
}

/** Format cents as EUR currency string */
function formatEur(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

export function FinancialsChart({ data }: FinancialsChartProps) {
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
      id: t('statistics.totalIncome'),
      color: '#22c55e',
      data: data.data_points.map((dp) => ({
        x: formatDateLabel(dp.date),
        y: centsToEur(dp.total_income),
      })),
    },
    {
      id: t('statistics.totalExpenses'),
      color: '#ef4444',
      data: data.data_points.map((dp) => ({
        x: formatDateLabel(dp.date),
        y: centsToEur(dp.total_expenses),
      })),
    },
    {
      id: t('statistics.balance'),
      color: '#3b82f6',
      data: data.data_points.map((dp) => ({
        x: formatDateLabel(dp.date),
        y: centsToEur(dp.balance),
      })),
    },
  ];

  return (
    <div className="h-[350px]">
      <ResponsiveLine
        data={chartData}
        margin={{ top: 20, right: 30, bottom: 50, left: 80 }}
        xScale={{ type: 'point' }}
        yScale={{ type: 'linear', min: 'auto', max: 'auto', stacked: false }}
        layers={[
          KitaYearBackgroundLayer,
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
          format: (v) =>
            Number(v).toLocaleString('de-DE', {
              style: 'currency',
              currency: 'EUR',
              maximumFractionDigits: 0,
            }),
        }}
        colors={['#22c55e', '#ef4444', '#3b82f6']}
        pointSize={8}
        pointColor={{ theme: 'background' }}
        pointBorderWidth={2}
        pointBorderColor={{ from: 'serieColor' }}
        useMesh={true}
        enableSlices="x"
        sliceTooltip={({ slice }) => {
          const idx = xLabels.indexOf(slice.points[0].data.xFormatted as string);
          const dp = idx >= 0 ? data.data_points[idx] : null;
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
                  {point.serieId}:{' '}
                  {Number(point.data.y).toLocaleString('de-DE', {
                    style: 'currency',
                    currency: 'EUR',
                  })}
                </div>
              ))}
              {dp && (
                <div
                  style={{
                    marginTop: 6,
                    paddingTop: 6,
                    borderTop: '1px solid hsl(var(--border))',
                    fontSize: 12,
                    opacity: 0.8,
                  }}
                >
                  <div>
                    {t('statistics.fundingIncome')}: {formatEur(dp.funding_income)}
                  </div>
                  {dp.funding_details?.map((fd) => (
                    <div key={`${fd.key}:${fd.value}`} style={{ paddingLeft: 12, opacity: 0.85 }}>
                      {fd.value}: {formatEur(fd.amount_cents)}
                    </div>
                  ))}
                  <div>
                    {t('statistics.budgetIncome')}: {formatEur(dp.budget_income)}
                  </div>
                  {dp.budget_item_details
                    ?.filter((bi) => bi.category === 'income')
                    .map((bi) => (
                      <div key={bi.name} style={{ paddingLeft: 12, opacity: 0.85 }}>
                        {bi.name}: {formatEur(bi.amount_cents)}
                      </div>
                    ))}
                  <div>
                    {t('statistics.grossSalary')}: {formatEur(dp.gross_salary)}
                  </div>
                  <div>
                    {t('statistics.employerCosts')}: {formatEur(dp.employer_costs)}
                  </div>
                  <div>
                    {t('statistics.budgetExpenses')}: {formatEur(dp.budget_expenses)}
                  </div>
                  {dp.budget_item_details
                    ?.filter((bi) => bi.category === 'expense')
                    .map((bi) => (
                      <div key={bi.name} style={{ paddingLeft: 12, opacity: 0.85 }}>
                        {bi.name}: {formatEur(bi.amount_cents)}
                      </div>
                    ))}
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
            itemWidth: 130,
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
