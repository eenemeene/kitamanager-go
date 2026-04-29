import { render, screen } from '@testing-library/react';
import { ContractPropertiesChart } from '../contract-properties-chart';
import type { ContractPropertiesDistributionResponse } from '@/lib/api/types';

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

describe('ContractPropertiesChart', () => {
  const mockData: ContractPropertiesDistributionResponse = {
    date: '2024-01-01',
    total_children: 30,
    properties: [
      { key: 'care_type', value: 'ganztag', count: 20, label: 'Ganztag' },
      { key: 'care_type', value: 'halbtag', count: 5, label: 'Halbtag' },
      { key: 'supplements', value: 'mss', count: 8, label: 'MSS' },
      { key: 'supplements', value: 'ndh', count: 12, label: 'NDH' },
    ],
  };

  it('renders the chart component', () => {
    render(<ContractPropertiesChart data={mockData} />);

    expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
  });

  it('displays total children count', () => {
    render(<ContractPropertiesChart data={mockData} />);

    expect(screen.getByText(/statistics\.totalChildren/)).toBeInTheDocument();
  });

  it('passes correct number of data points to chart', () => {
    render(<ContractPropertiesChart data={mockData} />);

    expect(screen.getByTestId('data-length')).toHaveTextContent('4');
  });

  it('passes value key to chart', () => {
    render(<ContractPropertiesChart data={mockData} />);

    expect(screen.getByTestId('keys')).toHaveTextContent('value');
  });

  it('sets aria label for accessibility', () => {
    render(<ContractPropertiesChart data={mockData} />);

    expect(screen.getByLabelText('statistics.contractProperties')).toBeInTheDocument();
  });

  it('handles empty properties', () => {
    const emptyData: ContractPropertiesDistributionResponse = {
      date: '2024-01-01',
      total_children: 0,
      properties: [],
    };

    render(<ContractPropertiesChart data={emptyData} />);

    expect(screen.getByTestId('data-length')).toHaveTextContent('0');
  });
});
