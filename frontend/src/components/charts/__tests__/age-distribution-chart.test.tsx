import { render, screen } from '@testing-library/react';
import { AgeDistributionChart } from '../age-distribution-chart';
import type { AgeDistributionResponse } from '@/lib/api/types';

// Mock Nivo's ResponsiveBar since it requires a DOM with dimensions
jest.mock('@nivo/bar', () => ({
  ResponsiveBar: ({
    data,
    keys,
    ariaLabel,
  }: {
    data: unknown[];
    keys: string[];
    ariaLabel: string;
  }) => (
    <div data-testid="bar-chart" aria-label={ariaLabel}>
      <span data-testid="data-length">{data.length}</span>
      <span data-testid="keys">{keys.join(',')}</span>
    </div>
  ),
}));

describe('AgeDistributionChart', () => {
  const mockData: AgeDistributionResponse = {
    date: '2024-01-01',
    distribution: [
      {
        age_label: '0',
        min_age: 0,
        max_age: 1,
        count: 5,
        male_count: 2,
        female_count: 2,
        diverse_count: 1,
      },
      {
        age_label: '1',
        min_age: 1,
        max_age: 2,
        count: 8,
        male_count: 4,
        female_count: 3,
        diverse_count: 1,
      },
      {
        age_label: '2',
        min_age: 2,
        max_age: 3,
        count: 10,
        male_count: 5,
        female_count: 4,
        diverse_count: 1,
      },
      {
        age_label: '3',
        min_age: 3,
        max_age: 4,
        count: 12,
        male_count: 6,
        female_count: 5,
        diverse_count: 1,
      },
      {
        age_label: '4',
        min_age: 4,
        max_age: 5,
        count: 9,
        male_count: 4,
        female_count: 4,
        diverse_count: 1,
      },
      {
        age_label: '5',
        min_age: 5,
        max_age: 6,
        count: 7,
        male_count: 3,
        female_count: 3,
        diverse_count: 1,
      },
      {
        age_label: '6+',
        min_age: 6,
        max_age: null,
        count: 4,
        male_count: 2,
        female_count: 2,
        diverse_count: 0,
      },
    ],
    total_count: 55,
  };

  it('renders the chart component', () => {
    render(<AgeDistributionChart data={mockData} />);

    expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
  });

  it('displays total children count', () => {
    render(<AgeDistributionChart data={mockData} />);

    expect(screen.getByText(/statistics\.totalChildren/)).toBeInTheDocument();
  });

  it('passes correct number of data points to chart', () => {
    render(<AgeDistributionChart data={mockData} />);

    expect(screen.getByTestId('data-length')).toHaveTextContent('7');
  });

  it('passes gender keys to chart', () => {
    render(<AgeDistributionChart data={mockData} />);

    const keys = screen.getByTestId('keys').textContent;
    expect(keys).toContain('gender.male');
    expect(keys).toContain('gender.female');
    expect(keys).toContain('gender.diverse');
  });

  it('sets aria label for accessibility', () => {
    render(<AgeDistributionChart data={mockData} />);

    expect(screen.getByLabelText('statistics.ageDistribution')).toBeInTheDocument();
  });

  it('handles empty distribution', () => {
    const emptyData: AgeDistributionResponse = {
      date: '2024-01-01',
      distribution: [],
      total_count: 0,
    };

    render(<AgeDistributionChart data={emptyData} />);

    expect(screen.getByTestId('data-length')).toHaveTextContent('0');
  });
});
