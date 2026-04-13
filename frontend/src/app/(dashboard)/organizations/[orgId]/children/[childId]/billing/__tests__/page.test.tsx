import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ChildBillingHistoryPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1', childId: '101' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string) => key;
    t.has = () => false;
    return t;
  },
  useLocale: () => 'en',
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getChildBillingHistory: jest.fn(),
  },
  getErrorMessage: jest.fn((_e: unknown, f: string) => f),
}));

const mockHistory = {
  child_id: 101,
  child_name: 'Max Mustermann',
  birthdate: '2020-03-10',
  voucher_numbers: ['GB-12345678901-02'],
  total_billed: 256000,
  total_calculated: 256000,
  total_difference: 0,
  entries: [
    {
      bill_id: 1,
      bill_from: '2025-01-01',
      bill_to: '2025-01-31',
      facility_name: 'Kita Sonnenschein',
      voucher_number: 'GB-12345678901-02',
      child_name: 'Mustermann, Max',
      birth_date: '03.20',
      age: 4,
      bill_total: 128000,
      calculated_total: 128000,
      difference: 0,
      status: 'match' as const,
      running_difference: 0,
      properties: [
        {
          key: 'care_type',
          value: 'ganztag',
          label: 'Full-Time',
          bill_amount: 128000,
          calculated_amount: 128000,
          difference: 0,
        },
      ],
      contract_id: 1,
    },
    {
      bill_id: 2,
      bill_from: '2025-02-01',
      bill_to: '2025-02-28',
      facility_name: 'Kita Sonnenschein',
      voucher_number: 'GB-12345678901-02',
      child_name: 'Mustermann, Max',
      birth_date: '03.20',
      age: 4,
      bill_total: 128000,
      calculated_total: 128000,
      difference: 0,
      status: 'match' as const,
      running_difference: 0,
      properties: [
        {
          key: 'care_type',
          value: 'ganztag',
          label: 'Full-Time',
          bill_amount: 128000,
          calculated_amount: 128000,
          difference: 0,
        },
      ],
      contract_id: 1,
    },
  ],
};

const mockEmptyHistory = {
  child_id: 101,
  child_name: 'Max Mustermann',
  birthdate: '2020-03-10',
  voucher_numbers: [],
  total_billed: 0,
  total_calculated: 0,
  total_difference: 0,
  entries: [],
};

const mockHistoryWithDifference = {
  child_id: 101,
  child_name: 'Max Mustermann',
  birthdate: '2020-03-10',
  voucher_numbers: ['GB-12345678901-02'],
  total_billed: 119900,
  total_calculated: 120000,
  total_difference: -100,
  entries: [
    {
      bill_id: 1,
      bill_from: '2025-01-01',
      bill_to: '2025-01-31',
      facility_name: 'Kita Sonnenschein',
      voucher_number: 'GB-12345678901-02',
      child_name: 'Mustermann, Max',
      birth_date: '03.20',
      age: 4,
      bill_total: 119900,
      calculated_total: 120000,
      difference: -100,
      status: 'difference' as const,
      running_difference: -100,
      properties: [
        {
          key: 'care_type',
          value: 'ganztag',
          label: 'Full-Time',
          bill_amount: 119900,
          calculated_amount: 120000,
          difference: -100,
        },
      ],
      contract_id: 1,
    },
  ],
};

describe('ChildBillingHistoryPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders billing history with entries', async () => {
    (apiClient.getChildBillingHistory as jest.Mock).mockResolvedValue(mockHistory);

    renderWithProviders(<ChildBillingHistoryPage />);

    await waitFor(() => {
      expect(screen.getAllByText('Max Mustermann').length).toBeGreaterThan(0);
    });

    // Voucher number appears in badge + each row
    expect(screen.getAllByText('GB-12345678901-02').length).toBeGreaterThan(0);

    // Two billing entries (facility name appears in each row)
    expect(screen.getAllByText('Kita Sonnenschein')).toHaveLength(2);
  });

  it('renders empty state when no entries', async () => {
    (apiClient.getChildBillingHistory as jest.Mock).mockResolvedValue(mockEmptyHistory);

    renderWithProviders(<ChildBillingHistoryPage />);

    await waitFor(() => {
      expect(screen.getByText('noBillingEntries')).toBeInTheDocument();
    });
  });

  it('renders no voucher numbers message', async () => {
    (apiClient.getChildBillingHistory as jest.Mock).mockResolvedValue(mockEmptyHistory);

    renderWithProviders(<ChildBillingHistoryPage />);

    await waitFor(() => {
      expect(screen.getByText('noVoucherNumbers')).toBeInTheDocument();
    });
  });

  it('shows difference status badge', async () => {
    (apiClient.getChildBillingHistory as jest.Mock).mockResolvedValue(mockHistoryWithDifference);

    renderWithProviders(<ChildBillingHistoryPage />);

    await waitFor(() => {
      expect(screen.getByText('statusDifference')).toBeInTheDocument();
    });
  });

  it('expands row to show property details on click', async () => {
    (apiClient.getChildBillingHistory as jest.Mock).mockResolvedValue(mockHistory);
    const user = userEvent.setup();

    renderWithProviders(<ChildBillingHistoryPage />);

    await waitFor(() => {
      expect(screen.getAllByText('Max Mustermann').length).toBeGreaterThan(0);
    });

    // Click first row to expand
    const firstRow = screen.getAllByText('Kita Sonnenschein')[0];
    await user.click(firstRow.closest('tr')!);

    // Property details should be visible
    await waitFor(() => {
      expect(screen.getByText('Full-Time')).toBeInTheDocument();
    });
  });

  it('calls API with correct parameters', async () => {
    (apiClient.getChildBillingHistory as jest.Mock).mockResolvedValue(mockHistory);

    renderWithProviders(<ChildBillingHistoryPage />);

    await waitFor(() => {
      expect(apiClient.getChildBillingHistory).toHaveBeenCalledWith(1, 101);
    });
  });
});
