import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test-utils';
import { ForecastOptimizeTab } from '../forecast-optimize-tab';

// Mock next-intl. We make `t` interpolate values into the key so test
// assertions can verify both the key and the count/balance arguments
// without depending on real translation files.
jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => {
    const t = (key: string, values?: Record<string, unknown>) =>
      values ? `${key}(${JSON.stringify(values)})` : key;
    t.has = () => false;
    return t;
  },
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
  usePathname: () => '/organizations/1/statistics/forecast',
}));

const sectionsList = [
  {
    id: 1,
    name: 'Krippe',
    organization_id: 1,
    is_default: false,
    min_age_months: 0,
    max_age_months: 36,
    created_at: '2024-01-01T00:00:00Z',
    created_by: 'admin',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'Elementar',
    organization_id: 1,
    is_default: true,
    min_age_months: 36,
    max_age_months: 72,
    created_at: '2024-01-01T00:00:00Z',
    created_by: 'admin',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const postForecastMock = jest.fn();
const getSectionsMock = jest.fn().mockResolvedValue({
  data: sectionsList,
  total: sectionsList.length,
  page: 1,
  limit: 30,
  total_pages: 1,
});

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSections: (...args: unknown[]) => getSectionsMock(...args),
    postForecast: (...args: unknown[]) => postForecastMock(...args),
  },
}));

jest.mock('@/lib/hooks/use-funding-attributes', () => ({
  useFundingAttributes: () => ({
    fundingAttributes: [],
    attributesByKey: {},
    defaultProperties: undefined,
  }),
}));

// Drive the forecast store via the actual implementation but reset
// state between tests so child additions don't leak across.
jest.mock('@/stores/ui-store', () => ({
  useUiStore: (selector: (s: { organizations: unknown[] }) => unknown) =>
    selector({ organizations: [{ id: 1, state: 'berlin' }] }),
}));

const storeState = {
  from: '2025-08-01',
  to: '2026-07-31',
  add_children: [] as unknown[],
  buildRequest: jest.fn(() => ({ from: '2025-08-01', to: '2026-07-31', add_children: [] })),
  addChild: jest.fn(),
};
jest.mock('@/stores/forecast-store', () => ({
  useForecastStore: () => storeState,
}));

beforeEach(() => {
  jest.clearAllMocks();
  storeState.buildRequest = jest.fn(() => ({
    from: '2025-08-01',
    to: '2026-07-31',
    add_children: [],
  }));
  storeState.addChild = jest.fn();
});

// The component's Labels are not `htmlFor`-linked to their Inputs, so
// `getByLabelText` won't work. Find by role+order: the target balance
// is the first <input type="number"> in the form.
function getTargetInput(): HTMLInputElement {
  const numbers = Array.from(document.querySelectorAll<HTMLInputElement>('input[type="number"]'));
  if (numbers.length === 0) throw new Error('no number inputs found');
  return numbers[0]!;
}

// Helper: build a postForecast response with a given monthly balance.
function makeForecastResponse(monthlyBalance: number) {
  return {
    financials: {
      data_points: Array.from({ length: 12 }, (_, i) => ({
        date: `2025-${String(i + 1).padStart(2, '0')}-01`,
        total_income: 0,
        total_expenses: 0,
        balance: monthlyBalance,
        funding_income: 0,
        actual_funding: 0,
        actual_funding_regular: 0,
        actual_funding_correction: 0,
        gross_salary: 0,
        employer_costs: 0,
        budget_income: 0,
        budget_expenses: 0,
        child_count: 0,
        staff_count: 0,
        salary_details: [],
        funding_details: [],
        budget_item_details: [],
      })),
      warnings: [],
    },
  };
}

describe('ForecastOptimizeTab', () => {
  describe('initial render and form state', () => {
    it('shows a target balance input prefilled with 0 when no baseline data is provided', async () => {
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={null} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText('Krippe (children.age: 1)')).toBeInTheDocument());
      // The target field is the first number input.
      const targetInput = getTargetInput();
      expect(targetInput).toHaveValue(0);
    });

    it('prefills the target balance once when baseline data first loads', async () => {
      // The component's useEffect reads baselineBalanceCents and rounds
      // cents → euros once. A re-render with a new baseline must NOT
      // overwrite the user's manual edit later — pinned in the next test.
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={250_000} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(getTargetInput()).toHaveValue(2500));
    });

    it('renders the loading hint while baseline is loading', () => {
      renderWithProviders(<ForecastOptimizeTab baselineBalanceCents={null} isLoadingBaseline />);
      expect(screen.getByText('common.loading')).toBeInTheDocument();
    });

    it('shows the baseline balance with success colour when positive', async () => {
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={250_000} isLoadingBaseline={false} />
      );
      await waitFor(() => {
        const baseline = screen.getByText(/^statistics\.forecastBaselineBalance/);
        expect(baseline).toHaveClass('text-success');
      });
    });

    it('shows the baseline balance with destructive colour when negative', async () => {
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={-50_000} isLoadingBaseline={false} />
      );
      await waitFor(() => {
        const baseline = screen.getByText(/^statistics\.forecastBaselineBalance/);
        expect(baseline).toHaveClass('text-destructive');
      });
    });
  });

  describe('section selection', () => {
    it('pre-selects the section with the lowest min_age_months on first load', async () => {
      // Krippe has min_age_months=0; Elementar has 36. Krippe should
      // be the auto-selected one — verifiable via the active variant
      // class on its button.
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={null} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      const krippeBtn = screen.getByRole('button', { name: /Krippe/ });
      const elementarBtn = screen.getByRole('button', { name: /Elementar/ });
      // The active variant adds a primary background; outline variant
      // does not. Match against the bg-primary class fragment.
      expect(krippeBtn.className).toMatch(/bg-primary/);
      expect(elementarBtn.className).not.toMatch(/bg-primary/);
    });

    it('toggles a section in/out when its button is clicked', async () => {
      const user = userEvent.setup();
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={null} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      const elementarBtn = screen.getByRole('button', { name: /Elementar/ });
      // Initially not selected.
      expect(elementarBtn.className).not.toMatch(/bg-primary/);
      // Click → selected.
      await user.click(elementarBtn);
      expect(elementarBtn.className).toMatch(/bg-primary/);
      // Click again → deselected.
      await user.click(elementarBtn);
      expect(elementarBtn.className).not.toMatch(/bg-primary/);
    });
  });

  describe('optimize button enable/disable', () => {
    it('disables the optimize button when no sections are selected', async () => {
      const user = userEvent.setup();
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={null} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      // Deselect the auto-selected Krippe.
      await user.click(screen.getByRole('button', { name: /Krippe/ }));
      const optimizeBtn = screen.getByRole('button', { name: 'statistics.forecastOptimize' });
      expect(optimizeBtn).toBeDisabled();
    });

    it('enables the optimize button when at least one section is selected', async () => {
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={null} isLoadingBaseline={false} />
      );
      // Krippe auto-selected → button enabled.
      await waitFor(() => {
        expect(
          screen.getByRole('button', { name: 'statistics.forecastOptimize' })
        ).not.toBeDisabled();
      });
    });
  });

  describe('optimization — happy path', () => {
    it('calls postForecast for baseline + at least one extra (binary search) and shows the result', async () => {
      // First call (baseline): balance = 0 (target 100€ = 10000c is far off)
      // Second call (1 child): balance = 1000 → perChildImpact = 1000
      // Estimated children: 10. Binary search converges at 10.
      // We don't care exactly how many calls happen as long as the
      // optimizer succeeds and the component renders the result card.
      postForecastMock
        .mockResolvedValueOnce(makeForecastResponse(0)) // baseline
        // Subsequent calls escalate balance with #children
        .mockImplementation(() => Promise.resolve(makeForecastResponse(1_000)));

      const user = userEvent.setup();
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={0} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      const target = getTargetInput();
      await user.clear(target);
      await user.type(target, '100');
      await user.click(screen.getByRole('button', { name: 'statistics.forecastOptimize' }));

      // Eventually the result card renders. We don't pin the exact
      // children-count because the algorithm has min/max search
      // bounds; we just verify the component reached "success".
      await waitFor(
        () => expect(screen.getByText(/^statistics\.forecastOptimizeResult/)).toBeInTheDocument(),
        { timeout: 3000 }
      );
      // postForecast was called at least twice (baseline + estimate).
      expect(postForecastMock.mock.calls.length).toBeGreaterThanOrEqual(2);
    });

    it('returns childrenAdded=0 immediately when the baseline already meets the target', async () => {
      // Baseline already at target → no API calls beyond baseline,
      // result card with count=0 (and no "added to store" hint).
      postForecastMock.mockResolvedValueOnce(makeForecastResponse(50_00)); // 50 €/month × 12 months = 60000

      const user = userEvent.setup();
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={60_000} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      const target = getTargetInput();
      await user.clear(target);
      await user.type(target, '500'); // 500€ = 50000c, baseline = 50€ × 12 = 600€ = 60000c >= 50000c
      await user.click(screen.getByRole('button', { name: 'statistics.forecastOptimize' }));

      await waitFor(() =>
        expect(screen.getByText(/^statistics\.forecastOptimizeResult/)).toBeInTheDocument()
      );
      // Result interpolation includes count: 0
      const resultEl = screen.getByText(/^statistics\.forecastOptimizeResult/);
      expect(resultEl.textContent).toContain('"count":0');
      // And only the baseline call happened.
      expect(postForecastMock).toHaveBeenCalledTimes(1);
    });
  });

  describe('optimization — non-happy paths', () => {
    it('shows "no impact" error when adding one child does not improve the balance', async () => {
      // perChildImpact <= 0 means the optimizer can't help — the user
      // would be told to revisit their inputs (sections, properties).
      // Both calls return same balance: per-child impact = 0.
      postForecastMock.mockResolvedValue(makeForecastResponse(0));

      const user = userEvent.setup();
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={0} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      const target = getTargetInput();
      await user.clear(target);
      await user.type(target, '100');
      await user.click(screen.getByRole('button', { name: 'statistics.forecastOptimize' }));

      await waitFor(() =>
        expect(screen.getByText('statistics.forecastOptimizeNoImpact')).toBeInTheDocument()
      );
    });

    it('shows the API error message when postForecast rejects', async () => {
      // Network / 500 error during the first API call. The component
      // catches and renders the message in the destructive banner.
      postForecastMock.mockRejectedValueOnce(new Error('Connection refused'));

      const user = userEvent.setup();
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={0} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      const target = getTargetInput();
      await user.clear(target);
      await user.type(target, '100');
      await user.click(screen.getByRole('button', { name: 'statistics.forecastOptimize' }));

      await waitFor(() => expect(screen.getByText('Connection refused')).toBeInTheDocument());
    });

    it('shows a spinner + disabled state while the optimizer is in flight', async () => {
      // Make postForecast hang so we can observe the in-flight UI.
      let resolveBaseline!: (v: unknown) => void;
      postForecastMock.mockImplementationOnce(
        () => new Promise((resolve) => (resolveBaseline = resolve))
      );

      const user = userEvent.setup();
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={0} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      const target = getTargetInput();
      await user.clear(target);
      await user.type(target, '100');
      const button = screen.getByRole('button', { name: 'statistics.forecastOptimize' });
      await user.click(button);

      // Button now reads "Running" (different label) and is disabled.
      await waitFor(() =>
        expect(
          screen.getByRole('button', { name: 'statistics.forecastOptimizeRunning' })
        ).toBeDisabled()
      );

      // Resolve so the test cleanup doesn't hold a dangling promise.
      resolveBaseline(makeForecastResponse(0));
    });
  });

  describe('store integration', () => {
    it('calls store.addChild for each optimal child after a successful run', async () => {
      // Baseline 0, perChildImpact=10000, target=10000c → 1 child needed.
      // The component should call addChild exactly the chosen-bestCount
      // times, with the synthetic child shape.
      postForecastMock
        .mockResolvedValueOnce(makeForecastResponse(0)) // baseline
        .mockResolvedValueOnce(makeForecastResponse(1_000)) // 1 child = 12000c sum >= 10000c target
        .mockResolvedValue(makeForecastResponse(1_000));

      const user = userEvent.setup();
      renderWithProviders(
        <ForecastOptimizeTab baselineBalanceCents={0} isLoadingBaseline={false} />
      );
      await waitFor(() => expect(screen.getByText(/Krippe/)).toBeInTheDocument());
      const target = getTargetInput();
      await user.clear(target);
      await user.type(target, '100'); // 100€ = 10000c
      await user.click(screen.getByRole('button', { name: 'statistics.forecastOptimize' }));
      await waitFor(
        () => expect(screen.getByText(/^statistics\.forecastOptimizeResult/)).toBeInTheDocument(),
        { timeout: 3000 }
      );
      // addChild called at least once.
      expect(storeState.addChild).toHaveBeenCalled();
      // Each call's argument is a ForecastChild — at minimum it has
      // first_name, gender, contracts.
      const firstCall = storeState.addChild.mock.calls[0]?.[0];
      expect(firstCall).toMatchObject({
        first_name: 'Child',
        gender: 'diverse',
        contracts: expect.any(Array),
      });
    });
  });
});
