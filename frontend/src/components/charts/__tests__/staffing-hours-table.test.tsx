import { screen } from '@testing-library/react';
import { StaffingHoursTable } from '../staffing-hours-table';
import { renderWithProviders } from '@/test-utils';
import type { StaffingHoursResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useLocale: () => 'de',
  useTranslations: () => (key: string) => key,
}));

function makeDataPoint(
  date: string,
  overrides: Partial<StaffingHoursResponse['data_points'][0]> = {}
): StaffingHoursResponse['data_points'][0] {
  return {
    date,
    required_hours: 312,
    available_hours: 340,
    child_count: 45,
    staff_count: 12,
    ...overrides,
  };
}

const threeMonthData: StaffingHoursResponse = {
  data_points: [
    makeDataPoint('2026-01-01'),
    makeDataPoint('2026-02-01'),
    makeDataPoint('2026-03-01'),
  ],
};

describe('StaffingHoursTable', () => {
  it('renders metric rows', () => {
    renderWithProviders(<StaffingHoursTable data={threeMonthData} />);

    expect(screen.getByText('staffingRequired')).toBeInTheDocument();
    expect(screen.getByText('staffingAvailable')).toBeInTheDocument();
    expect(screen.getByText('staffingBalance')).toBeInTheDocument();
    expect(screen.getByText('staffingBalancePercent')).toBeInTheDocument();
    expect(screen.getByText('childrenContractCount')).toBeInTheDocument();
    expect(screen.getByText('staffCount')).toBeInTheDocument();
  });

  it('renders month columns and average', () => {
    renderWithProviders(<StaffingHoursTable data={threeMonthData} />);

    expect(screen.getByText('Jan. 26')).toBeInTheDocument();
    expect(screen.getByText('Feb. 26')).toBeInTheDocument();
    expect(screen.getByText(/M(ä|ae)rz?\.?\s*26/)).toBeInTheDocument();
    expect(screen.getByText('average')).toBeInTheDocument();
  });

  it('computes balance correctly', () => {
    const data: StaffingHoursResponse = {
      data_points: [makeDataPoint('2026-01-01', { required_hours: 100, available_hours: 150 })],
    };

    renderWithProviders(<StaffingHoursTable data={data} />);

    // Balance = 150 - 100 = 50.00
    expect(screen.getAllByText('50,00').length).toBeGreaterThanOrEqual(1);
  });

  it('computes balance % correctly', () => {
    const data: StaffingHoursResponse = {
      data_points: [makeDataPoint('2026-01-01', { required_hours: 200, available_hours: 220 })],
    };

    renderWithProviders(<StaffingHoursTable data={data} />);

    // Balance % = ((220 - 200) / 200) * 100 = 10.0%
    expect(screen.getAllByText('10,0%').length).toBeGreaterThanOrEqual(1);
  });

  it('computes averages', () => {
    const data: StaffingHoursResponse = {
      data_points: [
        makeDataPoint('2026-01-01', {
          required_hours: 100,
          available_hours: 120,
          child_count: 40,
          staff_count: 10,
        }),
        makeDataPoint('2026-02-01', {
          required_hours: 200,
          available_hours: 240,
          child_count: 50,
          staff_count: 14,
        }),
      ],
    };

    renderWithProviders(<StaffingHoursTable data={data} />);

    // Avg required = (100+200)/2 = 150.00
    expect(screen.getByText('150,00')).toBeInTheDocument();
    // Avg available = (120+240)/2 = 180.00
    expect(screen.getByText('180,00')).toBeInTheDocument();
  });

  it('applies green color for positive balance and red for negative', () => {
    const data: StaffingHoursResponse = {
      data_points: [
        makeDataPoint('2026-01-01', { required_hours: 100, available_hours: 150 }),
        makeDataPoint('2026-02-01', { required_hours: 200, available_hours: 180 }),
      ],
    };

    renderWithProviders(<StaffingHoursTable data={data} />);

    // Positive balance: 50.00
    const positiveCell = screen.getByText('50,00').closest('td');
    expect(positiveCell?.className).toMatch(/text-success/);

    // Negative balance: -20.00
    const negativeCell = screen.getByText('-20,00').closest('td');
    expect(negativeCell?.className).toMatch(/text-destructive/);
  });

  it('handles empty data', () => {
    const data: StaffingHoursResponse = { data_points: [] };

    renderWithProviders(<StaffingHoursTable data={data} />);

    expect(screen.getByText('chartError')).toBeInTheDocument();
  });

  it('shows dash for zero required hours in balance %', () => {
    const data: StaffingHoursResponse = {
      data_points: [makeDataPoint('2026-01-01', { required_hours: 0, available_hours: 0 })],
    };

    renderWithProviders(<StaffingHoursTable data={data} />);

    // Balance % should show dash when required is 0 (division by zero)
    const cells = document.querySelectorAll('td.tabular-nums');
    const dashCells = Array.from(cells).filter((cell) => cell.textContent === '\u2013');
    expect(dashCells.length).toBeGreaterThan(0);
  });

  it('shows dash for zero hour values', () => {
    const data: StaffingHoursResponse = {
      data_points: [
        makeDataPoint('2026-01-01', {
          required_hours: 0,
          available_hours: 0,
          child_count: 0,
          staff_count: 0,
        }),
      ],
    };

    renderWithProviders(<StaffingHoursTable data={data} />);

    const cells = document.querySelectorAll('td.tabular-nums');
    const dashCells = Array.from(cells).filter((cell) => cell.textContent === '\u2013');
    // At minimum required, available, balance, balance %, children, staff cells should be dashes
    expect(dashCells.length).toBeGreaterThanOrEqual(6);
  });
});
