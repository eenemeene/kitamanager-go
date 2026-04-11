'use client';

import React, { useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { AlertTriangle, ChevronDown, ChevronRight } from 'lucide-react';
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

  const allPoints = data.data_points;

  const rawDates = allPoints.map((dp) => dp.date);
  const xLabels = rawDates.map(formatDateLabel);
  const kitaYearBands = useMemo(() => buildKitaYearBands(rawDates), [rawDates]);

  const chartData: BarDatum[] = useMemo(
    () =>
      allPoints.map((dp) => {
        const entry: BarDatum = {
          date: formatDateLabel(dp.date),
          [calculatedKey]: dp.funding_income / 100,
        };
        if (dp.actual_funding != null) {
          entry[actualKey] = dp.actual_funding / 100;
        }
        return entry;
      }),
    [allPoints, calculatedKey, actualKey]
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
  }, [todayLabel, tCommon]);

  const [expandedYears, setExpandedYears] = useState<Set<string>>(new Set());

  const toggleYear = (label: string) => {
    setExpandedYears((prev) => {
      const next = new Set(prev);
      if (next.has(label)) next.delete(label);
      else next.add(label);
      return next;
    });
  };

  // Per-Kita-year summary with monthly detail: always show calculated, actual only for months with bills
  const kitaYearSummary = useMemo(() => {
    const map = new Map<
      string,
      {
        calculatedTotal: number;
        calculatedWithBill: number;
        actual: number;
        actualMonths: number;
        totalMonths: number;
        months: {
          date: string;
          calculated: number;
          actual: number | null;
          difference: number | null;
        }[];
      }
    >();
    for (const dp of data.data_points) {
      const ky = kitaYearLabel(dp.date);
      const entry = map.get(ky) ?? {
        calculatedTotal: 0,
        calculatedWithBill: 0,
        actual: 0,
        actualMonths: 0,
        totalMonths: 0,
        months: [],
      };
      entry.totalMonths += 1;
      entry.calculatedTotal += dp.funding_income;
      const hasActual = dp.actual_funding != null;
      if (hasActual) {
        entry.calculatedWithBill += dp.funding_income;
        entry.actual += dp.actual_funding!;
        entry.actualMonths += 1;
      }
      entry.months.push({
        date: dp.date,
        calculated: dp.funding_income,
        actual: dp.actual_funding ?? null,
        difference: hasActual ? dp.actual_funding! - dp.funding_income : null,
      });
      map.set(ky, entry);
    }
    return Array.from(map.entries()).map(([label, v]) => ({
      label,
      calculatedTotal: v.calculatedTotal,
      calculatedWithBill: v.calculatedWithBill,
      actual: v.actual,
      difference: v.actual - v.calculatedWithBill,
      actualMonths: v.actualMonths,
      totalMonths: v.totalMonths,
      hasBills: v.actualMonths > 0,
      complete: v.actualMonths === v.totalMonths,
      months: v.months,
    }));
  }, [data]);

  const currentMonth = getCurrentMonthStart();
  const currentMonthDP = data.data_points.find((dp) => dp.date === currentMonth);
  const missingCurrentBill = currentMonthDP != null && currentMonthDP.actual_funding == null;

  if (allPoints.length === 0) {
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
            const dp = allPoints.find((d) => formatDateLabel(d.date) === indexValue);
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
              {kitaYearSummary.map((row) => {
                const isExpanded = expandedYears.has(row.label);
                return (
                  <React.Fragment key={row.label}>
                    <TableRow
                      className="hover:bg-muted/50 cursor-pointer"
                      onClick={() => toggleYear(row.label)}
                    >
                      <TableCell className="font-medium">
                        <div className="flex items-center gap-2">
                          {isExpanded ? (
                            <ChevronDown className="h-4 w-4 shrink-0" />
                          ) : (
                            <ChevronRight className="h-4 w-4 shrink-0" />
                          )}
                          {t('kitaYear', { year: row.label })}
                          {row.hasBills && !row.complete && (
                            <span className="inline-flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
                              <AlertTriangle className="h-3 w-3" />
                              {row.actualMonths}/{row.totalMonths} {t('fundingMonthsCovered')}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {formatEur(row.calculatedTotal)}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {row.hasBills ? formatEur(row.actual) : '\u2014'}
                      </TableCell>
                      <TableCell
                        className={`text-right font-medium tabular-nums ${
                          !row.hasBills
                            ? 'text-muted-foreground'
                            : row.difference >= 0
                              ? 'text-green-700 dark:text-green-400'
                              : 'text-red-700 dark:text-red-400'
                        }`}
                      >
                        {row.hasBills
                          ? `${row.difference >= 0 ? '+' : ''}${formatEur(row.difference)}`
                          : '\u2014'}
                      </TableCell>
                    </TableRow>
                    {isExpanded &&
                      row.months.map((m) => (
                        <TableRow key={m.date} className="bg-muted/30">
                          <TableCell className="pl-10 text-sm">{formatDateLabel(m.date)}</TableCell>
                          <TableCell className="text-right text-sm tabular-nums">
                            {formatEur(m.calculated)}
                          </TableCell>
                          <TableCell className="text-right text-sm tabular-nums">
                            {m.actual != null ? formatEur(m.actual) : '\u2014'}
                          </TableCell>
                          <TableCell
                            className={`text-right text-sm tabular-nums ${
                              m.difference == null
                                ? 'text-muted-foreground'
                                : m.difference >= 0
                                  ? 'text-green-700 dark:text-green-400'
                                  : 'text-red-700 dark:text-red-400'
                            }`}
                          >
                            {m.difference != null
                              ? `${m.difference >= 0 ? '+' : ''}${formatEur(m.difference)}`
                              : '\u2014'}
                          </TableCell>
                        </TableRow>
                      ))}
                  </React.Fragment>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
