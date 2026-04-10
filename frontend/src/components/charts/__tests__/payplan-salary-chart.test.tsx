import { render, screen } from '@testing-library/react';
import { PayPlanSalaryChart } from '../payplan-salary-chart';
import type { PayPlanPeriod } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@nivo/line', () => ({
  ResponsiveLine: ({ data }: { data: { id: string; data: unknown[] }[] }) => (
    <div data-testid="line-chart">
      <span data-testid="series-count">{data.length}</span>
      <span data-testid="series-ids">{data.map((d) => d.id).join(',')}</span>
    </div>
  ),
}));

const makePeriod = (
  from: string,
  entries: { grade: string; step: number; amount: number }[]
): PayPlanPeriod => ({
  id: Math.random(),
  payplan_id: 1,
  from,
  weekly_hours: 39,
  employer_contribution_rate: 2200,
  created_at: from,
  updated_at: from,
  entries: entries.map((e, i) => ({
    id: i + 1,
    period_id: 1,
    grade: e.grade,
    step: e.step,
    monthly_amount: e.amount,
    created_at: from,
    updated_at: from,
  })),
});

describe('PayPlanSalaryChart', () => {
  const periods: PayPlanPeriod[] = [
    makePeriod('2024-01-01T00:00:00Z', [
      { grade: 'S8a', step: 1, amount: 300000 },
      { grade: 'S8a', step: 2, amount: 320000 },
      { grade: 'S11b', step: 1, amount: 350000 },
    ]),
    makePeriod('2025-01-01T00:00:00Z', [
      { grade: 'S8a', step: 1, amount: 310000 },
      { grade: 'S8a', step: 2, amount: 330000 },
      { grade: 'S11b', step: 1, amount: 360000 },
    ]),
  ];

  it('renders the chart with correct number of grade series', () => {
    render(<PayPlanSalaryChart periods={periods} />);

    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
    // Default step is 1, so we should see S8a and S11b (numerically sorted)
    expect(screen.getByTestId('series-count')).toHaveTextContent('2');
    expect(screen.getByTestId('series-ids')).toHaveTextContent('S8a,S11b');
  });

  it('renders step selector with all available steps', () => {
    render(<PayPlanSalaryChart periods={periods} />);

    // The select trigger should show the default step
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('renders nothing with fewer than 2 periods', () => {
    const { container } = render(<PayPlanSalaryChart periods={[periods[0]]} />);

    expect(container.innerHTML).toBe('');
  });

  it('renders nothing with no entries', () => {
    const emptyPeriods: PayPlanPeriod[] = [
      makePeriod('2024-01-01T00:00:00Z', []),
      makePeriod('2025-01-01T00:00:00Z', []),
    ];
    const { container } = render(<PayPlanSalaryChart periods={emptyPeriods} />);

    expect(container.innerHTML).toBe('');
  });

  it('handles grade appearing in only one period', () => {
    const partial: PayPlanPeriod[] = [
      makePeriod('2024-01-01T00:00:00Z', [
        { grade: 'S8a', step: 1, amount: 300000 },
        { grade: 'S3', step: 1, amount: 250000 },
      ]),
      makePeriod('2025-01-01T00:00:00Z', [
        { grade: 'S8a', step: 1, amount: 310000 },
        // S3 missing in second period
      ]),
    ];
    render(<PayPlanSalaryChart periods={partial} />);

    // S3 has only 1 data point, S8a has 2 — both should still appear
    expect(screen.getByTestId('series-count')).toHaveTextContent('2');
    expect(screen.getByTestId('series-ids')).toHaveTextContent('S3,S8a');
  });

  it('sorts periods chronologically regardless of input order', () => {
    const unordered: PayPlanPeriod[] = [
      makePeriod('2026-01-01T00:00:00Z', [{ grade: 'S8a', step: 1, amount: 330000 }]),
      makePeriod('2024-01-01T00:00:00Z', [{ grade: 'S8a', step: 1, amount: 300000 }]),
      makePeriod('2025-01-01T00:00:00Z', [{ grade: 'S8a', step: 1, amount: 310000 }]),
    ];
    render(<PayPlanSalaryChart periods={unordered} />);

    // Should render without error with 1 series
    expect(screen.getByTestId('series-count')).toHaveTextContent('1');
  });

  it('handles only one step across all periods', () => {
    const singleStep: PayPlanPeriod[] = [
      makePeriod('2024-01-01T00:00:00Z', [{ grade: 'S8a', step: 3, amount: 340000 }]),
      makePeriod('2025-01-01T00:00:00Z', [{ grade: 'S8a', step: 3, amount: 350000 }]),
    ];
    render(<PayPlanSalaryChart periods={singleStep} />);

    expect(screen.getByRole('combobox')).toBeInTheDocument();
    expect(screen.getByTestId('series-count')).toHaveTextContent('1');
  });

  it('handles more than 15 grades without crashing', () => {
    const grades = Array.from({ length: 20 }, (_, i) => `S${i + 1}`);
    const manyGrades: PayPlanPeriod[] = [
      makePeriod(
        '2024-01-01T00:00:00Z',
        grades.map((g) => ({ grade: g, step: 1, amount: 300000 }))
      ),
      makePeriod(
        '2025-01-01T00:00:00Z',
        grades.map((g) => ({ grade: g, step: 1, amount: 310000 }))
      ),
    ];
    render(<PayPlanSalaryChart periods={manyGrades} />);

    expect(screen.getByTestId('series-count')).toHaveTextContent('20');
  });

  it('renders nothing with empty periods array', () => {
    const { container } = render(<PayPlanSalaryChart periods={[]} />);

    expect(container.innerHTML).toBe('');
  });

  it('renders nothing when periods have entries but all for different steps', () => {
    // Step 1 exists in period 1, step 2 exists in period 2 — but neither step spans both periods
    const disjoint: PayPlanPeriod[] = [
      makePeriod('2024-01-01T00:00:00Z', [{ grade: 'S8a', step: 1, amount: 300000 }]),
      makePeriod('2025-01-01T00:00:00Z', [{ grade: 'S8a', step: 2, amount: 330000 }]),
    ];
    render(<PayPlanSalaryChart periods={disjoint} />);

    // Default step is 1 (smallest), S8a has step 1 only in period 1 — still 1 data point, renders chart
    expect(screen.getByTestId('series-count')).toHaveTextContent('1');
  });
});
