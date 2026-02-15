import { render, screen } from '@testing-library/react';
import { MonthlyContractChart } from '../monthly-contract-chart';
import type { StaffingHoursResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

// Mock Nivo's ResponsiveLine since it requires a DOM with dimensions
jest.mock('@nivo/line', () => ({
  ResponsiveLine: ({ data }: { data: { id: string; data: unknown[] }[] }) => (
    <div data-testid="line-chart">
      <span data-testid="series-count">{data.length}</span>
      <span data-testid="series-ids">{data.map((d) => d.id).join(',')}</span>
      <span data-testid="points-count">{data[0]?.data.length || 0}</span>
    </div>
  ),
}));

describe('MonthlyContractChart', () => {
  const mockData: StaffingHoursResponse = {
    data_points: [
      {
        date: '2025-01-01',
        required_hours: 100,
        available_hours: 120,
        child_count: 10,
        staff_count: 5,
      },
      {
        date: '2025-02-01',
        required_hours: 110,
        available_hours: 120,
        child_count: 12,
        staff_count: 5,
      },
      {
        date: '2025-03-01',
        required_hours: 120,
        available_hours: 130,
        child_count: 15,
        staff_count: 6,
      },
    ],
  };

  it('renders the chart component', () => {
    render(<MonthlyContractChart data={mockData} />);

    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
  });

  it('renders a single series for child count', () => {
    render(<MonthlyContractChart data={mockData} />);

    expect(screen.getByTestId('series-count')).toHaveTextContent('1');
  });

  it('has one data point per month', () => {
    render(<MonthlyContractChart data={mockData} />);

    expect(screen.getByTestId('points-count')).toHaveTextContent('3');
  });

  it('handles empty data', () => {
    const emptyData: StaffingHoursResponse = { data_points: [] };

    render(<MonthlyContractChart data={emptyData} />);

    expect(screen.getByTestId('series-count')).toHaveTextContent('1');
    expect(screen.getByTestId('points-count')).toHaveTextContent('0');
  });
});
