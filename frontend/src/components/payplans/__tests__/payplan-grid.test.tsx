import { screen } from '@testing-library/react';
import { PayPlanGrid } from '../payplan-grid';
import { renderWithProviders } from '@/test-utils';
import type { PayPlanPeriod } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

const ts = '2024-01-01T00:00:00Z';

function makePeriod(
  overrides: Partial<PayPlanPeriod> & { entries?: PayPlanPeriod['entries'] }
): PayPlanPeriod {
  return {
    id: 1,
    payplan_id: 1,
    from: '2024-01-01',
    weekly_hours: 39,
    employer_contribution_rate: 2200,
    created_at: ts,
    updated_at: ts,
    ...overrides,
  };
}

function makeEntry(
  id: number,
  grade: string,
  step: number,
  monthly_amount: number,
  step_min_years: number
) {
  return {
    id,
    period_id: 1,
    grade,
    step,
    monthly_amount,
    step_min_years,
    created_at: ts,
    updated_at: ts,
  };
}

describe('PayPlanGrid', () => {
  it('returns null when no entries', () => {
    const period = makePeriod({ entries: [] });
    const { container } = renderWithProviders(<PayPlanGrid period={period} />);
    expect(container.querySelector('table')).not.toBeInTheDocument();
  });

  it('returns null when entries is undefined', () => {
    const period = makePeriod({});
    const { container } = renderWithProviders(<PayPlanGrid period={period} />);
    expect(container.querySelector('table')).not.toBeInTheDocument();
  });

  it('renders grid with grades and steps', () => {
    const period = makePeriod({
      entries: [
        makeEntry(1, 'S8a', 1, 350000, 0),
        makeEntry(2, 'S8a', 2, 370000, 1),
        makeEntry(3, 'S3', 1, 280000, 0),
      ],
    });
    renderWithProviders(<PayPlanGrid period={period} />);

    // Grades should appear
    expect(screen.getByText('S8a')).toBeInTheDocument();
    expect(screen.getByText('S3')).toBeInTheDocument();

    // Steps should appear in header (prefixed with translated "step" key)
    expect(screen.getByText(/payPlans\.step 1/)).toBeInTheDocument();
    expect(screen.getByText(/payPlans\.step 2/)).toBeInTheDocument();
  });

  it('renders formatted currency amounts', () => {
    const period = makePeriod({
      entries: [makeEntry(1, 'S8a', 1, 350000, 0)],
    });
    renderWithProviders(<PayPlanGrid period={period} />);
    // 350000 cents = 3500.00 EUR, formatted with formatCurrency
    expect(screen.getByText(/3[.,]500/)).toBeInTheDocument();
  });

  it('renders percentage increase between steps', () => {
    const period = makePeriod({
      entries: [makeEntry(1, 'S8a', 1, 100000, 0), makeEntry(2, 'S8a', 2, 105000, 1)],
    });
    renderWithProviders(<PayPlanGrid period={period} />);
    // 5% increase
    expect(screen.getByText('↗5.0%')).toBeInTheDocument();
  });

  it('sorts grades by numeric part descending', () => {
    const period = makePeriod({
      entries: [makeEntry(1, 'S3', 1, 280000, 0), makeEntry(2, 'S8a', 1, 350000, 0)],
    });
    renderWithProviders(<PayPlanGrid period={period} />);

    const rows = screen.getAllByRole('row');
    // First data row (index 1, after header) should be S8a (grade 8 > grade 3)
    expect(rows[1]).toHaveTextContent('S8a');
    expect(rows[2]).toHaveTextContent('S3');
  });

  it('renders step minimum years in header', () => {
    const period = makePeriod({
      entries: [makeEntry(1, 'S8a', 1, 350000, 0), makeEntry(2, 'S8a', 2, 370000, 3)],
    });
    renderWithProviders(<PayPlanGrid period={period} />);
    expect(screen.getByText('(3y)')).toBeInTheDocument();
  });
});
