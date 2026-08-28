import { screen, within } from '@testing-library/react';
import { MonthlyContractTable } from '../monthly-contract-table';
import { renderWithProviders } from '@/test-utils';
import { expectNoA11yViolations } from '@/test-a11y';
import type { OccupancyResponse, StaffingHoursResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useLocale: () => 'de',
  useTranslations: () => (key: string) => key,
}));

const staffing: StaffingHoursResponse = {
  data_points: [
    {
      date: '2026-01-01',
      child_count: 61,
      required_hours: 300,
      available_hours: 320,
      staff_count: 11,
    },
    {
      date: '2026-02-01',
      child_count: 63,
      required_hours: 310,
      available_hours: 320,
      staff_count: 11,
    },
  ],
};

const occupancy: OccupancyResponse = {
  age_groups: [
    { key: 'nest', label: 'Nest', value: 'nest' },
    { key: 'gross', label: 'Große', value: 'gross' },
  ],
  care_types: [],
  supplement_types: [],
  data_points: [
    {
      date: '2026-01-01',
      total: 61,
      by_supplement: {},
      // two care types under one group: the table sums them, as the chart does
      by_age_and_care_type: { Nest: { ganztag: 20, halbtag: 5 }, Große: { ganztag: 36 } },
    },
    {
      date: '2026-02-01',
      total: 63,
      by_supplement: {},
      by_age_and_care_type: { Nest: { ganztag: 27 }, Große: { ganztag: 36 } },
    },
  ],
} as unknown as OccupancyResponse;

describe('MonthlyContractTable', () => {
  it('renders a row per month with the contract count', () => {
    renderWithProviders(<MonthlyContractTable data={staffing} occupancy={occupancy} />);
    const rows = screen.getAllByRole('row');
    expect(rows).toHaveLength(3); // header + two months
    expect(within(rows[1]).getByText('61')).toBeInTheDocument();
    expect(within(rows[2]).getByText('63')).toBeInTheDocument();
  });

  it('sums the care types nested under each age group', () => {
    renderWithProviders(<MonthlyContractTable data={staffing} occupancy={occupancy} />);
    // January Nest is 20 + 5
    expect(within(screen.getAllByRole('row')[1]).getByText('25')).toBeInTheDocument();
  });

  // The occupancy query is separate from the staffing one, so the table has to
  // render before it arrives rather than blanking the page.
  it('degrades to the total when occupancy has not loaded', () => {
    renderWithProviders(<MonthlyContractTable data={staffing} />);
    expect(screen.getAllByRole('row')).toHaveLength(3);
    expect(screen.queryByText('Nest')).not.toBeInTheDocument();
    expect(screen.getByText('61')).toBeInTheDocument();
  });

  it('has no accessibility violations', async () => {
    const { container } = renderWithProviders(
      <MonthlyContractTable data={staffing} occupancy={occupancy} />
    );
    await expectNoA11yViolations(container);
  });
});
