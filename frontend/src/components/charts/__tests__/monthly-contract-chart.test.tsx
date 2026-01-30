import { render, screen } from '@testing-library/react';
import { MonthlyContractChart } from '../monthly-contract-chart';
import type { ChildrenContractCountByMonthResponse } from '@/lib/api/types';

// Mock Nivo's ResponsiveLine since it requires a DOM with dimensions
jest.mock('@nivo/line', () => ({
  ResponsiveLine: ({ data }: { data: { id: string; data: unknown[] }[] }) => (
    <div data-testid="line-chart">
      <span data-testid="series-count">{data.length}</span>
      <span data-testid="series-ids">{data.map((d) => d.id).join(',')}</span>
      <span data-testid="points-per-series">{data[0]?.data.length || 0}</span>
    </div>
  ),
}));

describe('MonthlyContractChart', () => {
  const mockData: ChildrenContractCountByMonthResponse = {
    period: {
      start: '2024-01-01',
      end: '2025-12-31',
    },
    years: [
      {
        year: 2024,
        counts: [10, 12, 15, 14, 16, 18, 20, 22, 25, 24, 23, 22],
      },
      {
        year: 2025,
        counts: [22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44],
      },
    ],
  };

  it('renders the chart component', () => {
    render(<MonthlyContractChart data={mockData} />);

    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
  });

  it('passes correct number of series to chart', () => {
    render(<MonthlyContractChart data={mockData} />);

    expect(screen.getByTestId('series-count')).toHaveTextContent('2');
  });

  it('uses year as series id', () => {
    render(<MonthlyContractChart data={mockData} />);

    const seriesIds = screen.getByTestId('series-ids').textContent;
    expect(seriesIds).toContain('2024');
    expect(seriesIds).toContain('2025');
  });

  it('has 12 data points per series (one per month)', () => {
    render(<MonthlyContractChart data={mockData} />);

    expect(screen.getByTestId('points-per-series')).toHaveTextContent('12');
  });

  it('handles single year data', () => {
    const singleYearData: ChildrenContractCountByMonthResponse = {
      period: { start: '2024-01-01', end: '2024-12-31' },
      years: [
        {
          year: 2024,
          counts: [10, 12, 15, 14, 16, 18, 20, 22, 25, 24, 23, 22],
        },
      ],
    };

    render(<MonthlyContractChart data={singleYearData} />);

    expect(screen.getByTestId('series-count')).toHaveTextContent('1');
    expect(screen.getByTestId('series-ids')).toHaveTextContent('2024');
  });

  it('handles empty data', () => {
    const emptyData: ChildrenContractCountByMonthResponse = {
      period: { start: '2024-01-01', end: '2024-12-31' },
      years: [],
    };

    render(<MonthlyContractChart data={emptyData} />);

    expect(screen.getByTestId('series-count')).toHaveTextContent('0');
  });

  it('handles year with missing counts', () => {
    const partialData: ChildrenContractCountByMonthResponse = {
      period: { start: '2024-01-01', end: '2024-12-31' },
      years: [
        {
          year: 2024,
          counts: [10, 12, 15], // Only 3 months
        },
      ],
    };

    render(<MonthlyContractChart data={partialData} />);

    // Should still create 12 data points (filling with 0 for missing)
    expect(screen.getByTestId('points-per-series')).toHaveTextContent('12');
  });
});
