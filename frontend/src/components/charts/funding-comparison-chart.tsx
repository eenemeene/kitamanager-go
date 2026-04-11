'use client';

import { useMemo } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { AlertTriangle } from 'lucide-react';
import { ResponsiveBar } from '@nivo/bar';
import type { BarDatum, BarCustomLayerProps } from '@nivo/bar';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { ExportableChart } from './exportable-chart';
import type { FinancialResponse } from '@/lib/api/types';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { buildKitaYearBands, formatDateLabel, kitaYearLabel, chartTheme } from './chart-utils';
import { toLocalDateString, getCurrentMonthStart } from '@/lib/utils/formatting';

interface FundingComparisonChartProps {
  data: FinancialResponse;
}

type BandScale = ((v: string) => number | undefined) & { bandwidth(): number };

function formatEur(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

export function FundingComparisonChart({ data }: FundingComparisonChartProps) {
  const t = useTranslations('statistics');
  const tCommon = useTranslations('common');
  const params = useParams();
  const orgId = params.orgId;

  const calculatedKey = t('fundingCalculated');
  const actualKey = t('fundingActual');

  // Only include months that have actual funding data
  const filteredPoints = useMemo(
    () => data.data_points.filter((dp) => dp.actual_funding != null),
    [data]
  );

  const rawDates = filteredPoints.map((dp) => dp.date);
  const xLabels = rawDates.map(formatDateLabel);
  const kitaYearBands = useMemo(() => buildKitaYearBands(rawDates), [rawDates]);

  const chartData: BarDatum[] = useMemo(
    () =>
      filteredPoints.map((dp) => ({
        date: formatDateLabel(dp.date),
        [calculatedKey]: dp.funding_income / 100,
        [actualKey]: (dp.actual_funding ?? 0) / 100,
      })),
    [filteredPoints, calculatedKey, actualKey]
  );

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
                        {t('kitaYear', { year: band.label })}
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
            {tCommon('today')}
          </text>
        </g>
      );
    };
  }, [todayLabel, t]);

  // Per-Kita-year summary: only compare months that have actual bill data
  const kitaYearSummary = useMemo(() => {
    const map = new Map<
      string,
      { calculated: number; actual: number; actualMonths: number; totalMonths: number }
    >();
    for (const dp of data.data_points) {
      const ky = kitaYearLabel(dp.date);
      const entry = map.get(ky) ?? { calculated: 0, actual: 0, actualMonths: 0, totalMonths: 0 };
      entry.totalMonths += 1;
      if (dp.actual_funding != null) {
        // Only include calculated amount for months where we have a bill
        entry.calculated += dp.funding_income;
        entry.actual += dp.actual_funding;
        entry.actualMonths += 1;
      }
      map.set(ky, entry);
    }
    return Array.from(map.entries())
      .filter(([, v]) => v.actualMonths > 0)
      .map(([label, v]) => ({
        label,
        calculated: v.calculated,
        actual: v.actual,
        difference: v.actual - v.calculated,
        actualMonths: v.actualMonths,
        totalMonths: v.totalMonths,
        complete: v.actualMonths === v.totalMonths,
      }));
  }, [data]);

  const currentMonth = getCurrentMonthStart();
  const currentMonthDP = data.data_points.find((dp) => dp.date === currentMonth);
  const missingCurrentBill = currentMonthDP != null && currentMonthDP.actual_funding == null;

  if (chartData.length === 0) {
    return (
      <div className="space-y-4">
        <p className="text-muted-foreground">{t('fundingNoDataYet')}</p>
        {missingCurrentBill && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              {t('fundingBillMissing')}{' '}
              <Link
                href={`/organizations/${orgId}/government-funding-bills`}
                className="font-medium underline"
              >
                {t('fundingBillUploadLink')}
              </Link>
            </AlertDescription>
          </Alert>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {missingCurrentBill && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            {t('fundingBillMissing')}{' '}
            <Link
              href={`/organizations/${orgId}/government-funding-bills`}
              className="font-medium underline"
            >
              {t('fundingBillUploadLink')}
            </Link>
          </AlertDescription>
        </Alert>
      )}
      <ExportableChart filename="funding-comparison" className="h-[500px]">
        <ResponsiveBar
          data={chartData}
          keys={[calculatedKey, actualKey]}
          indexBy="date"
          groupMode="grouped"
          margin={{ top: 40, right: 30, bottom: 130, left: 90 }}
          padding={0.3}
          innerPadding={2}
          valueScale={{ type: 'linear' }}
          colors={['#3b82f6', '#f59e0b']}
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
          enableLabel={false}
          tooltip={({ indexValue, id, value, color }) => {
            const dp = filteredPoints.find((d) => formatDateLabel(d.date) === indexValue);
            const diff =
              dp && dp.actual_funding != null ? dp.actual_funding - dp.funding_income : null;
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
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    marginTop: 4,
                  }}
                >
                  <span
                    style={{
                      width: 10,
                      height: 10,
                      borderRadius: '50%',
                      background: color,
                      display: 'inline-block',
                    }}
                  />
                  {id}: {formatEur((value as number) * 100)}
                </div>
                {diff != null && (
                  <div
                    style={{
                      marginTop: 4,
                      color: diff >= 0 ? '#22c55e' : '#ef4444',
                      fontSize: 12,
                    }}
                  >
                    {t('fundingDifference')}: {diff >= 0 ? '+' : ''}
                    {formatEur(diff)}
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
              itemsSpacing: 20,
              itemDirection: 'left-to-right',
              itemWidth: 150,
              itemHeight: 20,
              itemOpacity: 0.85,
              symbolSize: 12,
              symbolShape: 'circle',
            },
          ]}
          role="application"
          ariaLabel={t('fundingActualVsCalculated')}
          theme={chartTheme}
        />
      </ExportableChart>
      {kitaYearSummary.length > 0 && (
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('kitaYearCol')}</TableHead>
                <TableHead className="text-right">{t('fundingCalculated')}</TableHead>
                <TableHead className="text-right">{t('fundingActual')}</TableHead>
                <TableHead className="text-right">{t('fundingDifference')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {kitaYearSummary.map((row) => (
                <TableRow key={row.label}>
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                      {t('kitaYear', { year: row.label })}
                      {!row.complete && (
                        <span className="inline-flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
                          <AlertTriangle className="h-3 w-3" />
                          {row.actualMonths}/{row.totalMonths} {t('fundingMonthsCovered')}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {formatEur(row.calculated)}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">{formatEur(row.actual)}</TableCell>
                  <TableCell
                    className={`text-right font-medium tabular-nums ${
                      row.difference >= 0
                        ? 'text-green-700 dark:text-green-400'
                        : 'text-red-700 dark:text-red-400'
                    }`}
                  >
                    {row.difference >= 0 ? '+' : ''}
                    {formatEur(row.difference)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
