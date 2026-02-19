/** Minimal props used by custom Nivo layers — avoids complex generic constraints. */
interface ChartLayerProps {
  xScale: (value: string) => number;
  innerHeight: number;
  innerWidth: number;
}

export interface KitaYearBand {
  label: string;
  startIdx: number;
  endIdx: number;
}

/** Returns the Kita year label for a given date (Aug–Jul). e.g. August 2024 → "24/25" */
export function kitaYearLabel(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  const month = date.getMonth(); // 0-indexed
  const year = date.getFullYear();
  const startYear = month >= 7 ? year : year - 1; // Aug (7) starts a new Kita year
  const sy = String(startYear).slice(2);
  const ey = String(startYear + 1).slice(2);
  return `${sy}/${ey}`;
}

/** Groups consecutive data point indices by their Kita year */
export function buildKitaYearBands(dates: string[]): KitaYearBand[] {
  if (dates.length === 0) return [];
  const bands: KitaYearBand[] = [];
  let currentLabel = kitaYearLabel(dates[0]);
  let startIdx = 0;
  for (let i = 1; i < dates.length; i++) {
    const label = kitaYearLabel(dates[i]);
    if (label !== currentLabel) {
      bands.push({ label: currentLabel, startIdx, endIdx: i - 1 });
      currentLabel = label;
      startIdx = i;
    }
  }
  bands.push({ label: currentLabel, startIdx, endIdx: dates.length - 1 });
  return bands;
}

/** Format a date string as "Jan 25", "Feb 25", etc. */
export function formatDateLabel(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  return date.toLocaleDateString('en-US', { month: 'short', year: '2-digit' });
}

/** Creates a Nivo custom layer that draws alternating background bands per Kita year */
export function createKitaYearBackgroundLayer(
  kitaYearBands: KitaYearBand[],
  xLabels: string[],
  kitaYearText: (label: string) => string
) {
  return function KitaYearBg({ xScale, innerHeight, innerWidth }: ChartLayerProps) {
    const scale = xScale;
    const step = xLabels.length > 1 ? scale(xLabels[1]) - scale(xLabels[0]) : innerWidth;

    return (
      <g>
        {kitaYearBands.map((band, i) => {
          const x0 = scale(xLabels[band.startIdx]) - step / 2;
          const x1 = scale(xLabels[band.endIdx]) + step / 2;
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
                    {/* Horizontal spanning line */}
                    <line
                      x1={clampedX0 + 4}
                      x2={clampedX1 - 4}
                      y1={bracketY}
                      y2={bracketY}
                      stroke="currentColor"
                      strokeWidth={1}
                      opacity={0.3}
                    />
                    {/* Left tick */}
                    <line
                      x1={clampedX0 + 4}
                      x2={clampedX0 + 4}
                      y1={bracketY - tickH}
                      y2={bracketY}
                      stroke="currentColor"
                      strokeWidth={1}
                      opacity={0.3}
                    />
                    {/* Right tick */}
                    <line
                      x1={clampedX1 - 4}
                      x2={clampedX1 - 4}
                      y1={bracketY - tickH}
                      y2={bracketY}
                      stroke="currentColor"
                      strokeWidth={1}
                      opacity={0.3}
                    />
                    {/* Center connector down to label */}
                    <line
                      x1={midX}
                      x2={midX}
                      y1={bracketY}
                      y2={bracketY + 4}
                      stroke="currentColor"
                      strokeWidth={1}
                      opacity={0.3}
                    />
                    {/* Label */}
                    <text
                      x={midX}
                      y={labelY}
                      textAnchor="middle"
                      fontSize={11}
                      fontWeight={500}
                      fill="currentColor"
                      opacity={0.5}
                    >
                      {kitaYearText(band.label)}
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
}

/** Creates a today marker config for Nivo line charts */
export function createTodayMarker(todayLabel: string, legendText: string) {
  return {
    axis: 'x' as const,
    value: todayLabel,
    lineStyle: {
      stroke: 'hsl(var(--foreground))',
      strokeWidth: 1,
      strokeDasharray: '4 4',
    },
    legend: legendText,
    legendPosition: 'top' as const,
    textStyle: {
      fill: 'hsl(var(--muted-foreground))',
      fontSize: 11,
    },
  };
}

/** Shared Nivo theme for charts that use CSS variables */
export const chartTheme = {
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
  crosshair: {
    line: {
      stroke: 'hsl(var(--foreground))',
      strokeWidth: 1,
      strokeOpacity: 0.35,
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
};
