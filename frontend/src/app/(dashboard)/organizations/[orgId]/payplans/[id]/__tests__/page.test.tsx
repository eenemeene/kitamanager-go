import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import PayplanDetailPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1', id: '1' }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/components/charts/payplan-salary-chart', () => ({
  PayPlanSalaryChart: () => <div data-testid="salary-chart" />,
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getPayPlan: jest.fn(),
    getPayPlanExportUrl: jest.fn().mockReturnValue('/export'),
    createPayPlanPeriod: jest.fn(),
    updatePayPlanPeriod: jest.fn(),
    deletePayPlanPeriod: jest.fn(),
    createPayPlanEntry: jest.fn(),
    updatePayPlanEntry: jest.fn(),
    deletePayPlanEntry: jest.fn(),
  },
  getErrorMessage: jest.fn((_e: unknown, f: string) => f),
}));

describe('PayplanDetailPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getPayPlan as jest.Mock).mockImplementation(() => new Promise(() => {}));
  });

  it('renders loading state', () => {
    const { container } = renderWithProviders(<PayplanDetailPage />);
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('clicking the entry pencil opens the edit dialog with values pre-populated', async () => {
    (apiClient.getPayPlan as jest.Mock).mockResolvedValue({
      id: 1,
      organization_id: 1,
      name: 'TVöD',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      periods: [
        {
          id: 10,
          payplan_id: 1,
          from: '2024-01-01T00:00:00Z',
          to: null,
          weekly_hours: 39,
          employer_contribution_rate: 2200,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 100,
              period_id: 10,
              grade: 'S8a',
              step: 3,
              monthly_amount: 350000,
              step_min_years: null,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
        },
      ],
    });

    renderWithProviders(<PayplanDetailPage />);

    // Wait for initial load, then switch to "panels" view where individual
    // entries get their per-row edit/delete buttons (the default "table" view
    // is the compact PayPlanGrid summary, no per-row actions).
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'payPlans.viewPanels' })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole('button', { name: 'payPlans.viewPanels' }));

    // Now the entry row with the new edit button should be visible.
    await waitFor(() => expect(screen.getByText('S8a')).toBeInTheDocument());
    const editButtons = screen.getAllByRole('button', { name: 'common.edit' });
    expect(editButtons.length).toBeGreaterThan(0);
    fireEvent.click(editButtons[0]);

    // Dialog title should reflect edit (was "addEntry" before this fix).
    const dialogTitle = await screen.findByText('payPlans.editEntry');
    const dialog = dialogTitle.closest('[role="dialog"]') as HTMLElement;
    expect(dialog).not.toBeNull();

    // Form should be pre-populated from the row.
    const gradeInput = within(dialog).getByLabelText('payPlans.gradeLabel') as HTMLInputElement;
    expect(gradeInput.value).toBe('S8a');
    const stepInput = within(dialog).getByLabelText('payPlans.stepLabel') as HTMLInputElement;
    expect(stepInput.value).toBe('3');
    const amountInput = within(dialog).getByLabelText(
      'payPlans.monthlyAmountInEuros'
    ) as HTMLInputElement;
    // monthly_amount is cents on the wire (350000), pre-fill is in euros.
    expect(amountInput.value).toBe('3500');
  });
});
