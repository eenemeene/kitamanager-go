import { screen, waitFor } from '@testing-library/react';
import EmployeeContractsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1', employeeId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getEmployee: jest.fn(),
    getEmployeeContracts: jest.fn(),
    createEmployeeContract: jest.fn(),
    updateEmployeeContract: jest.fn(),
    deleteEmployeeContract: jest.fn(),
  },
  getErrorMessage: jest.fn((e: unknown, f: string) => f),
}));

const mockEmployee = {
  id: 1,
  first_name: 'John',
  last_name: 'Doe',
  gender: 'male',
  birthdate: '1990-01-15T00:00:00Z',
  organization_id: 1,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const mockContracts = [
  {
    id: 1,
    employee_id: 1,
    from: '2020-01-01T00:00:00Z',
    to: null,
    staff_category: 'qualified',
    grade: 'S8a',
    step: 3,
    weekly_hours: 39,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

describe('EmployeeContractsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders page title with employee name', async () => {
    (apiClient.getEmployee as jest.Mock).mockResolvedValue(mockEmployee);
    (apiClient.getEmployeeContracts as jest.Mock).mockResolvedValue(mockContracts);

    renderWithProviders(<EmployeeContractsPage />);

    await waitFor(() => {
      expect(screen.getByText('John Doe')).toBeInTheDocument();
    });

    const historyLabels = screen.getAllByText('employees.contractHistory');
    expect(historyLabels.length).toBeGreaterThanOrEqual(1);
  });

  it('shows loading skeleton', async () => {
    (apiClient.getEmployee as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );
    (apiClient.getEmployeeContracts as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    renderWithProviders(<EmployeeContractsPage />);

    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('displays contracts in table', async () => {
    (apiClient.getEmployee as jest.Mock).mockResolvedValue(mockEmployee);
    (apiClient.getEmployeeContracts as jest.Mock).mockResolvedValue(mockContracts);

    renderWithProviders(<EmployeeContractsPage />);

    await waitFor(() => {
      expect(screen.getByText('employees.staffCategory.qualified')).toBeInTheDocument();
    });

    expect(screen.getByText('S8a / 3')).toBeInTheDocument();
    expect(screen.getByText('39h')).toBeInTheDocument();
  });

  it('shows no contracts message when empty', async () => {
    (apiClient.getEmployee as jest.Mock).mockResolvedValue(mockEmployee);
    (apiClient.getEmployeeContracts as jest.Mock).mockResolvedValue([]);

    renderWithProviders(<EmployeeContractsPage />);

    await waitFor(() => {
      expect(screen.getByText('John Doe')).toBeInTheDocument();
    });

    const noContractsMessages = screen.getAllByText('employees.noContractsFound');
    expect(noContractsMessages.length).toBeGreaterThanOrEqual(1);
  });

  it('renders back button', async () => {
    (apiClient.getEmployee as jest.Mock).mockResolvedValue(mockEmployee);
    (apiClient.getEmployeeContracts as jest.Mock).mockResolvedValue(mockContracts);

    renderWithProviders(<EmployeeContractsPage />);

    await waitFor(() => {
      expect(screen.getByText('John Doe')).toBeInTheDocument();
    });

    // Back button is a ghost button with ArrowLeft icon
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThanOrEqual(1);
  });
});
