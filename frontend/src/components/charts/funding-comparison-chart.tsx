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
import type {
  FinancialResponse,
  FundingComparisonResponse,
  FundingComparisonSummary,
} from '@/lib/api/types';
import { FundingDeficitAnalysis } from './funding-deficit-analysis';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { TooltipProvider } from '@/components/ui/tooltip';
import { HeaderWithTooltip } from '@/components/ui/header-with-tooltip';
import { buildKitaYearBands, useDateLabel, kitaYearLabel, chartTheme } from './chart-utils';
import { getCurrentMonthStart } from '@/lib/utils/formatting';
import { useFormatters } from '@/hooks/use-formatters';
import { todayBerlinString } from '@/lib/utils/contracts';

interface FundingComparisonChartProps {
  data: FinancialResponse;
  compareData?: Map<string, FundingComparisonResponse>;
  compareSummaries?: Map<string, FundingComparisonSummary>;
  /** When true, all kita year rows and deficit analysis sections are expanded. Used for print/PDF export. */
  forceExpanded?: boolean;
}

type BandScale = ((v: string) => number | undefined) & { bandwidth(): number };

export function FundingComparisonChart({
  data,
  compareData,
  compareSummaries,
  forceExpanded = false,
}: FundingComparisonChartProps) {
  const t = useTranslations('statistics');
  const formatDateLabel = useDateLabel();
  const fmt = useFormatters();
  const formatEur = (cents: number) => fmt.currency(cents);
  const tCommon = useTranslations('common');
  const params = useParams();
  const orgId = params.orgId;

  const calculatedKey = t('fundingCalculated');
  const actualRegularKey = t('fundingActualRegular');
  const actualCorrectionKey = t('fundingActualCorrection');

  const allPoints = data.data_points;

  const rawDates = allPoints.map((dp) => dp.date ?? '');
  const xLabels = rawDates.map(formatDateLabel);
  const kitaYearBands = useMemo(() => buildKitaYearBands(rawDates), [rawDates]);

  const chartData: BarDatum[] = useMemo(
    () =>
      allPoints.map((dp) => {
        const entry: BarDatum = {
          date: formatDateLabel(dp.date ?? ''),
          [calculatedKey]: (dp.funding_income ?? 0) / 100,
        };
        if (dp.actual_funding_regular != null) {
          entry[actualRegularKey] = dp.actual_funding_regular / 100;
        }
        if (dp.actual_funding_correction != null && dp.actual_funding_correction !== 0) {
          entry[actualCorrectionKey] = dp.actual_funding_correction / 100;
        }
        return entry;
      }),
    [allPoints, calculatedKey, actualRegularKey, actualCorrectionKey, formatDateLabel]
  );

  const todayStr = todayBerlinString();
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
                        style={{ cursor: 'help' }}
                      >
                        <title>{t('kitaYearTooltip')}</title>
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

  // Per-Kita-year summary with monthly detail
  const kitaYearSummary = useMemo(() => {
    const map = new Map<
      string,
      {
        calculatedTotal: number;
        calculatedWithBill: number;
        regular: number;
        correction: number;
        actualMonths: number;
        totalMonths: number;
        months: {
          date: string;
          calculated: number;
          regular: number | null;
          correction: number | null;
          difference: number | null;
          billOnlyCount: number | null;
          billOnlyAmount: number | null;
          calcOnlyCount: number | null;
          calcOnlyAmount: number | null;
        }[];
      }
    >();
    for (const dp of data.data_points) {
      const dpDate = dp.date ?? '';
      const dpFundingIncome = dp.funding_income ?? 0;
      const ky = kitaYearLabel(dpDate);
      const entry = map.get(ky) ?? {
        calculatedTotal: 0,
        calculatedWithBill: 0,
        regular: 0,
        correction: 0,
        actualMonths: 0,
        totalMonths: 0,
        months: [],
      };
      entry.totalMonths += 1;
      entry.calculatedTotal += dpFundingIncome;
      const hasActual = dp.actual_funding != null;
      if (hasActual) {
        entry.calculatedWithBill += dpFundingIncome;
        entry.regular += dp.actual_funding_regular ?? 0;
        entry.correction += dp.actual_funding_correction ?? 0;
        entry.actualMonths += 1;
      }
      const comp = compareData?.get(dpDate);
      entry.months.push({
        date: dpDate,
        calculated: dpFundingIncome,
        regular: dp.actual_funding_regular ?? null,
        correction: dp.actual_funding_correction ?? null,
        difference: hasActual ? (dp.actual_funding_regular ?? 0) - dpFundingIncome : null,
        billOnlyCount: comp?.bill_only_count ?? null,
        billOnlyAmount: comp?.bill_only_amount ?? null,
        calcOnlyCount: comp?.calc_only_count ?? null,
        calcOnlyAmount: comp?.calc_only_amount ?? null,
      });
      map.set(ky, entry);
    }
    return Array.from(map.entries()).map(([label, v]) => ({
      label,
      calculatedTotal: v.calculatedTotal,
      calculatedWithBill: v.calculatedWithBill,
      regular: v.regular,
      correction: v.correction,
      difference: v.regular + v.correction - v.calculatedWithBill,
      actualMonths: v.actualMonths,
      totalMonths: v.totalMonths,
      hasBills: v.actualMonths > 0,
      complete: v.actualMonths === v.totalMonths,
      months: v.months,
    }));
  }, [data, compareData]);

  // Match each Kita year to the best-overlapping compare summary window
  const kitaYearDeficitMap = useMemo(() => {
    if (!compareSummaries) return new Map<string, FundingComparisonSummary>();
    const result = new Map<string, FundingComparisonSummary>();
    for (const row of kitaYearSummary) {
      if (!row.hasBills) continue;
      let bestKey: string | undefined;
      let bestOverlap = 0;
      for (const [key] of compareSummaries) {
        const [wFrom, wTo] = key.split(':');
        const overlap = row.months.filter((m) => m.date >= wFrom && m.date <= wTo).length;
        if (overlap > bestOverlap) {
          bestOverlap = overlap;
          bestKey = key;
        }
      }
      if (bestKey) result.set(row.label, compareSummaries.get(bestKey)!);
    }
    return result;
  }, [compareSummaries, kitaYearSummary]);

  // In print/PDF mode, expand all kita year rows automatically
  const effectiveExpandedYears = useMemo(() => {
    if (forceExpanded && kitaYearSummary.length > 0) {
      return new Set(kitaYearSummary.map((row) => row.label));
    }
    return expandedYears;
  }, [forceExpanded, kitaYearSummary, expandedYears]);

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
          keys={[calculatedKey, actualRegularKey, actualCorrectionKey]}
          indexBy="date"
          groupMode="grouped"
          margin={{ top: 40, right: 30, bottom: 130, left: 90 }}
          padding={0.3}
          innerPadding={2}
          valueScale={{ type: 'linear' }}
          colors={['#3b82f6', '#f59e0b', '#ef4444']}
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
              fmt.number(Number(v), {
                style: 'currency',
                currency: 'EUR',
                maximumFractionDigits: 0,
              }),
          }}
          enableLabel={false}
          tooltip={({ indexValue, id, value, color }) => {
            const dp = allPoints.find((d) => formatDateLabel(d.date ?? '') === indexValue);
            const diff =
              dp && dp.actual_funding_regular != null
                ? dp.actual_funding_regular - (dp.funding_income ?? 0)
                : null;
            const comp = dp ? compareData?.get(dp.date ?? '') : undefined;
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
                {comp && comp.bill_only_count > 0 && (
                  <div style={{ marginTop: 2, fontSize: 12, color: '#f59e0b' }}>
                    {t('fundingBillOnly')}:{' '}
                    {t('fundingChildCount', { count: comp.bill_only_count })} (
                    {formatEur(comp.bill_only_amount)})
                  </div>
                )}
                {comp && comp.calc_only_count > 0 && (
                  <div style={{ marginTop: 2, fontSize: 12, color: '#3b82f6' }}>
                    {t('fundingCalcOnly')}:{' '}
                    {t('fundingChildCount', { count: comp.calc_only_count })} (
                    {formatEur(comp.calc_only_amount)})
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
          role="img"
          ariaLabel={t('fundingActualVsCalculated')}
          theme={chartTheme}
        />
      </ExportableChart>
      {kitaYearSummary.length > 0 && (
        <TooltipProvider>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('kitaYearCol')}</TableHead>
                  <TableHead className="text-right">
                    <HeaderWithTooltip
                      label={t('fundingCalculated')}
                      tooltip={t('fundingCalculatedTooltip')}
                    />
                  </TableHead>
                  <TableHead className="text-right">
                    <HeaderWithTooltip
                      label={t('fundingActualRegular')}
                      tooltip={t('fundingRegularTooltip')}
                    />
                  </TableHead>
                  <TableHead className="text-right">
                    <HeaderWithTooltip
                      label={t('fundingActualCorrection')}
                      tooltip={t('fundingCorrectionsTooltip')}
                    />
                  </TableHead>
                  <TableHead className="text-right">
                    <HeaderWithTooltip
                      label={t('fundingDifference')}
                      tooltip={t('fundingDifferenceTooltip')}
                    />
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {kitaYearSummary.map((row) => {
                  const isExpanded = effectiveExpandedYears.has(row.label);
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
                              <span className="text-warning inline-flex items-center gap-1 text-xs">
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
                          {row.hasBills ? formatEur(row.regular) : '\u2014'}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {row.hasBills ? formatEur(row.correction) : '\u2014'}
                        </TableCell>
                        <TableCell
                          className={`text-right font-medium tabular-nums ${
                            !row.hasBills
                              ? 'text-muted-foreground'
                              : row.difference >= 0
                                ? 'text-success'
                                : 'text-destructive'
                          }`}
                        >
                          {row.hasBills
                            ? `${row.difference >= 0 ? '+' : ''}${formatEur(row.difference)}`
                            : '\u2014'}
                        </TableCell>
                      </TableRow>
                      {isExpanded &&
                        row.months.map((m) => {
                          const hasMismatch =
                            (m.billOnlyCount != null && m.billOnlyCount > 0) ||
                            (m.calcOnlyCount != null && m.calcOnlyCount > 0);
                          return (
                            <React.Fragment key={m.date}>
                              <TableRow className="bg-muted/30">
                                <TableCell className="pl-10 text-sm">
                                  {formatDateLabel(m.date)}
                                </TableCell>
                                <TableCell className="text-right text-sm tabular-nums">
                                  {formatEur(m.calculated)}
                                </TableCell>
                                <TableCell className="text-right text-sm tabular-nums">
                                  {m.regular != null ? formatEur(m.regular) : '\u2014'}
                                </TableCell>
                                <TableCell className="text-right text-sm tabular-nums">
                                  {m.correction != null ? formatEur(m.correction) : '\u2014'}
                                </TableCell>
                                <TableCell
                                  className={`text-right text-sm tabular-nums ${
                                    m.difference == null
                                      ? 'text-muted-foreground'
                                      : m.difference >= 0
                                        ? 'text-success'
                                        : 'text-destructive'
                                  }`}
                                >
                                  {m.difference != null
                                    ? `${m.difference >= 0 ? '+' : ''}${formatEur(m.difference)}`
                                    : '\u2014'}
                                </TableCell>
                              </TableRow>
                              {hasMismatch && (
                                <TableRow className="bg-muted/20">
                                  <TableCell colSpan={5} className="py-1 pl-14">
                                    <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-0.5 text-xs">
                                      {m.billOnlyCount != null && m.billOnlyCount > 0 && (
                                        <span className="text-warning">
                                          {t('fundingBillOnly')}:{' '}
                                          {t('fundingChildCount', { count: m.billOnlyCount })} (
                                          {formatEur(m.billOnlyAmount ?? 0)})
                                        </span>
                                      )}
                                      {m.calcOnlyCount != null && m.calcOnlyCount > 0 && (
                                        <span className="text-info">
                                          {t('fundingCalcOnly')}:{' '}
                                          {t('fundingChildCount', { count: m.calcOnlyCount })} (
                                          {formatEur(m.calcOnlyAmount ?? 0)})
                                        </span>
                                      )}
                                    </div>
                                  </TableCell>
                                </TableRow>
                              )}
                            </React.Fragment>
                          );
                        })}
                      {kitaYearDeficitMap.has(row.label) && (
                        <FundingDeficitAnalysis
                          summary={kitaYearDeficitMap.get(row.label)!}
                          orgId={orgId!}
                          forceExpanded={forceExpanded}
                        />
                      )}
                    </React.Fragment>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        </TooltipProvider>
      )}
    </div>
  );
}
