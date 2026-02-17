'use client';

import { useMemo, useState, useCallback } from 'react';
import { useTranslations } from 'next-intl';
import { ResponsiveBar } from '@nivo/bar';
import type { BarDatum, BarCustomLayerProps } from '@nivo/bar';
import { line, curveMonotoneX } from 'd3-shape';
import type { FinancialResponse, FinancialDataPoint } from '@/lib/api/types';
import { buildKitaYearBands, formatDateLabel, chartTheme } from './chart-utils';

interface FinancialsChartProps {
  data: FinancialResponse;
}

function centsToEur(cents: number): number {
  return Math.round(cents) / 100;
}

function formatEur(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

type BandScale = ((v: string) => number | undefined) & { bandwidth(): number };

export function FinancialsChart({ data }: FinancialsChartProps) {
  const t = useTranslations();
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);
  const [tooltipPos, setTooltipPos] = useState<{ x: number; y: number }>({ x: 0, y: 0 });

  const fundingKey = t('statistics.fundingIncome');
  const budgetIncomeKey = t('statistics.budgetIncome');
  const grossSalaryKey = t('statistics.grossSalary');
  const employerCostsKey = t('statistics.employerCosts');
  const budgetExpensesKey = t('statistics.budgetExpenses');
  const balanceLabel = t('statistics.balance');

  const rawDates = data.data_points.map((dp) => dp.date);
  const xLabels = rawDates.map(formatDateLabel);
  const kitaYearBands = useMemo(() => buildKitaYearBands(rawDates), [rawDates]);

  const chartData: BarDatum[] = data.data_points.map((dp) => ({
    date: formatDateLabel(dp.date),
    [fundingKey]: centsToEur(dp.funding_income),
    [budgetIncomeKey]: centsToEur(dp.budget_income),
    [grossSalaryKey]: -centsToEur(dp.gross_salary),
    [employerCostsKey]: -centsToEur(dp.employer_costs),
    [budgetExpensesKey]: -centsToEur(dp.budget_expenses),
    _balance: centsToEur(dp.balance),
    _total_income: centsToEur(dp.total_income),
    _total_expenses: centsToEur(dp.total_expenses),
    _idx: data.data_points.indexOf(dp),
  }));

  const keys = [fundingKey, budgetIncomeKey, grossSalaryKey, employerCostsKey, budgetExpensesKey];
  const colors = ['#22c55e', '#14b8a6', '#ef4444', '#f97316', '#f59e0b'];

  const todayStr = new Date().toISOString().slice(0, 10);
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
                {/* Vertical separator line at kita year boundary */}
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
                {/* Spanning bracket with kita year label below the x-axis */}
                {(() => {
                  const bracketY = innerHeight + 48;
                  const tickH = 4;
                  const midX = clampedX0 + width / 2;
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

  const handleBalanceEnter = useCallback((idx: number, screenX: number, screenY: number) => {
    setHoveredIdx(idx);
    setTooltipPos({ x: screenX, y: screenY });
  }, []);

  const handleBalanceLeave = useCallback(() => {
    setHoveredIdx(null);
  }, []);

  const BalanceLine = useMemo(() => {
    return function BalanceLineLayer({ xScale, yScale }: BarCustomLayerProps<BarDatum>) {
      const scale = xScale as unknown as BandScale;
      const bw = scale.bandwidth();
      const yFn = yScale as unknown as (v: number) => number;

      const points = chartData.map((d) => ({
        x: (scale(d.date as string) ?? 0) + bw / 2,
        y: yFn(d._balance as number),
        balance: d._balance as number,
      }));

      const pathGen = line<(typeof points)[0]>()
        .x((d) => d.x)
        .y((d) => d.y)
        .curve(curveMonotoneX);

      const pathD = pathGen(points);

      return (
        <g>
          {pathD && <path d={pathD} fill="none" stroke="#3b82f6" strokeWidth={2} />}
          {points.map((p, i) => (
            <circle
              key={i}
              cx={p.x}
              cy={p.y}
              r={4}
              fill="#3b82f6"
              stroke="hsl(var(--background))"
              strokeWidth={2}
            />
          ))}
          {/* Invisible wider hit areas for hover */}
          {points.map((p, i) => (
            <circle
              key={`hit-${i}`}
              cx={p.x}
              cy={p.y}
              r={12}
              fill="transparent"
              style={{ cursor: 'pointer' }}
              onMouseEnter={(e) => handleBalanceEnter(i, e.clientX, e.clientY)}
              onMouseLeave={handleBalanceLeave}
            />
          ))}
        </g>
      );
    };
  }, [chartData, handleBalanceEnter, handleBalanceLeave]);

  const BalanceLegend = useMemo(() => {
    // 5 bar legend items: each 130px wide + 4px spacing = 134px per item
    const totalBarLegendWidth = 5 * 134;
    const balanceLegendWidth = 100;
    const totalLegendWidth = totalBarLegendWidth + balanceLegendWidth;

    return function BalanceLegendLayer({ innerWidth }: BarCustomLayerProps<BarDatum>) {
      const legendStartX = (innerWidth - totalLegendWidth) / 2;
      const balanceX = legendStartX + totalBarLegendWidth;

      return (
        <g transform={`translate(${balanceX}, -55)`}>
          <circle cx={6} cy={10} r={6} fill="#3b82f6" />
          <text x={16} y={14} fontSize={12} fill="hsl(var(--foreground))" opacity={0.85}>
            {balanceLabel}
          </text>
        </g>
      );
    };
  }, [balanceLabel]);

  const hoveredDp: FinancialDataPoint | null =
    hoveredIdx !== null ? data.data_points[hoveredIdx] : null;

  return (
    <div className="relative h-[400px]">
      <ResponsiveBar
        data={chartData}
        keys={keys}
        indexBy="date"
        margin={{ top: 60, right: 30, bottom: 80, left: 80 }}
        padding={0.3}
        groupMode="stacked"
        minValue="auto"
        maxValue="auto"
        colors={colors}
        layers={[
          KitaYearBackground,
          'grid',
          'axes',
          'bars',
          TodayMarker,
          BalanceLine,
          'markers',
          'legends',
          BalanceLegend,
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
        tooltip={({ id, value, indexValue, color }) => (
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
              {id}:{' '}
              {Math.abs(Number(value)).toLocaleString('de-DE', {
                style: 'currency',
                currency: 'EUR',
              })}
            </div>
            {id === employerCostsKey && (
              <div style={{ fontSize: 11, opacity: 0.7, marginTop: 2 }}>
                {t('statistics.employerCostsTooltip')}
              </div>
            )}
          </div>
        )}
        legends={[
          {
            dataFrom: 'keys',
            anchor: 'top',
            direction: 'row',
            justify: false,
            translateX: -50,
            translateY: -55,
            itemsSpacing: 4,
            itemDirection: 'left-to-right',
            itemWidth: 130,
            itemHeight: 20,
            itemOpacity: 0.85,
            symbolSize: 12,
            symbolShape: 'circle',
          },
        ]}
        role="application"
        ariaLabel={t('statistics.financialOverview')}
        theme={chartTheme}
      />
      {/* Balance point tooltip */}
      {hoveredDp && (
        <div
          style={{
            position: 'fixed',
            left: tooltipPos.x + 12,
            top: tooltipPos.y - 12,
            background: 'hsl(var(--background))',
            color: 'hsl(var(--foreground))',
            border: '1px solid hsl(var(--border))',
            borderRadius: '6px',
            padding: '9px 12px',
            fontSize: 13,
            pointerEvents: 'none',
            zIndex: 50,
            whiteSpace: 'nowrap',
          }}
        >
          <strong>{xLabels[hoveredIdx!]}</strong>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
            <span
              style={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                background: '#3b82f6',
                display: 'inline-block',
              }}
            />
            {balanceLabel}: {formatEur(hoveredDp.balance)}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
            <span
              style={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                background: '#22c55e',
                display: 'inline-block',
              }}
            />
            {t('statistics.totalIncome')}: {formatEur(hoveredDp.total_income)}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
            <span
              style={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                background: '#ef4444',
                display: 'inline-block',
              }}
            />
            {t('statistics.totalExpenses')}: {formatEur(hoveredDp.total_expenses)}
          </div>
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
              {t('statistics.fundingIncome')}: {formatEur(hoveredDp.funding_income)}
            </div>
            {hoveredDp.funding_details?.map((fd) => (
              <div key={`${fd.key}:${fd.value}`} style={{ paddingLeft: 12, opacity: 0.85 }}>
                {fd.value}: {formatEur(fd.amount_cents)}
              </div>
            ))}
            <div>
              {t('statistics.budgetIncome')}: {formatEur(hoveredDp.budget_income)}
            </div>
            {hoveredDp.budget_item_details
              ?.filter((bi) => bi.category === 'income')
              .map((bi) => (
                <div key={bi.name} style={{ paddingLeft: 12, opacity: 0.85 }}>
                  {bi.name}: {formatEur(bi.amount_cents)}
                </div>
              ))}
            <div>
              {t('statistics.grossSalary')}: {formatEur(hoveredDp.gross_salary)}
            </div>
            <div title={t('statistics.employerCostsTooltip')}>
              {t('statistics.employerCosts')}: {formatEur(hoveredDp.employer_costs)}
              <span style={{ fontSize: 11, opacity: 0.7, marginLeft: 4 }}>(?)</span>
            </div>
            <div>
              {t('statistics.budgetExpenses')}: {formatEur(hoveredDp.budget_expenses)}
            </div>
            {hoveredDp.budget_item_details
              ?.filter((bi) => bi.category === 'expense')
              .map((bi) => (
                <div key={bi.name} style={{ paddingLeft: 12, opacity: 0.85 }}>
                  {bi.name}: {formatEur(bi.amount_cents)}
                </div>
              ))}
          </div>
        </div>
      )}
    </div>
  );
}
