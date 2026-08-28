import { screen, within } from '@testing-library/react';
import { ContractPropertiesTable } from '../contract-properties-table';
import { renderWithProviders } from '@/test-utils';
import { expectNoA11yViolations } from '@/test-a11y';
import type { ContractPropertiesDistributionResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useLocale: () => 'de',
  useTranslations: () => (key: string) => key,
}));

const data: ContractPropertiesDistributionResponse = {
  date: '2026-08-28',
  total_children: 40,
  properties: [
    { key: 'care_type', value: 'ganztag', label: 'Ganztag (bis 9h)', count: 30 },
    { key: 'care_type', value: 'halbtag', label: 'Halbtag (bis 5h)', count: 10 },
  ],
};

describe('ContractPropertiesTable', () => {
  it('renders a row per property with its count', () => {
    renderWithProviders(<ContractPropertiesTable data={data} />);
    expect(screen.getByText('Ganztag (bis 9h)')).toBeInTheDocument();
    expect(screen.getByText('Halbtag (bis 5h)')).toBeInTheDocument();
  });

  // The share is the column the chart cannot express: a bar is comparable to
  // the other bars, not to the number of children overall.
  it('shows each property as a share of all children', () => {
    renderWithProviders(<ContractPropertiesTable data={data} />);
    const rows = screen.getAllByRole('row');
    expect(within(rows[1]).getByText(/75/)).toBeInTheDocument();
    expect(within(rows[2]).getByText(/25/)).toBeInTheDocument();
  });

  // An organisation with no children would otherwise divide by zero and render
  // NaN into the column.
  it('does not divide by zero when there are no children', () => {
    renderWithProviders(
      <ContractPropertiesTable
        data={{ ...data, total_children: 0, properties: [data.properties[0]] }}
      />
    );
    expect(screen.getByText('—')).toBeInTheDocument();
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument();
  });

  it('falls back to key and value when a property has no label', () => {
    renderWithProviders(
      <ContractPropertiesTable
        data={{ ...data, properties: [{ key: 'ndh', value: 'yes', label: '', count: 4 }] }}
      />
    );
    expect(screen.getByText('ndh: yes')).toBeInTheDocument();
  });

  it('has no accessibility violations', async () => {
    const { container } = renderWithProviders(<ContractPropertiesTable data={data} />);
    await expectNoA11yViolations(container);
  });
});
