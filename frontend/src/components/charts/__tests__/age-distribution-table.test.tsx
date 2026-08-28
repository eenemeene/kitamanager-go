import { screen, within } from '@testing-library/react';
import { AgeDistributionTable } from '../age-distribution-table';
import { renderWithProviders } from '@/test-utils';
import { expectNoA11yViolations } from '@/test-a11y';
import type { AgeDistributionResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useLocale: () => 'de',
  useTranslations: () => (key: string) => key,
}));

const data: AgeDistributionResponse = {
  date: '2026-08-28',
  total_count: 12,
  distribution: [
    {
      age_label: '1',
      min_age: 1,
      max_age: 2,
      count: 5,
      male_count: 2,
      female_count: 3,
      diverse_count: 0,
    },
    {
      age_label: '2',
      min_age: 2,
      max_age: 3,
      count: 4,
      male_count: 1,
      female_count: 2,
      diverse_count: 1,
    },
    // the open-ended bucket has no max_age, which is what makes it open-ended
    { age_label: '6+', min_age: 6, count: 3, male_count: 3, female_count: 0, diverse_count: 0 },
  ],
};

describe('AgeDistributionTable', () => {
  it('renders a row per age bucket with its gender split', () => {
    renderWithProviders(<AgeDistributionTable data={data} />);
    const rows = screen.getAllByRole('row');
    // header + three buckets + totals
    expect(rows).toHaveLength(5);
    expect(within(rows[1]).getByText('2')).toBeInTheDocument();
    expect(within(rows[1]).getByText('3')).toBeInTheDocument();
  });

  // "6+" is a bucket rather than an age, so it must not be fed through the
  // "{age} years" message the numeric labels use.
  it('labels the open-ended bucket separately', () => {
    renderWithProviders(<AgeDistributionTable data={data} />);
    expect(screen.getByText('statistics.ageSixPlus')).toBeInTheDocument();
  });

  it('totals each gender column', () => {
    renderWithProviders(<AgeDistributionTable data={data} />);
    const totalRow = screen.getAllByRole('row').at(-1)!;
    // male 2+1+3, female 3+2+0, diverse 0+1+0, all 5+4+3
    expect(within(totalRow).getByText('6')).toBeInTheDocument();
    expect(within(totalRow).getByText('5')).toBeInTheDocument();
    expect(within(totalRow).getByText('1')).toBeInTheDocument();
    expect(within(totalRow).getByText('12')).toBeInTheDocument();
  });

  it('names the table for a screen reader without repeating it on screen', () => {
    const { container } = renderWithProviders(<AgeDistributionTable data={data} />);
    const caption = container.querySelector('caption');
    expect(caption).toHaveTextContent('statistics.ageDistributionTable');
    expect(caption).toHaveClass('sr-only');
  });

  it('has no accessibility violations', async () => {
    const { container } = renderWithProviders(<AgeDistributionTable data={data} />);
    await expectNoA11yViolations(container);
  });
});
