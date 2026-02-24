import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import StaffingPrintPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getStaffingHours: jest.fn(),
    getEmployeeStaffingHours: jest.fn(),
    getSections: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => fallback),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/stores/ui-store', () => ({
  useUiStore: () => ({
    organizations: [{ id: 1, name: 'Test Kita' }],
    fetchOrganizations: jest.fn(),
  }),
}));

const mockStaffingData = {
  data_points: [
    {
      date: '2026-01-01',
      required_hours: 312,
      available_hours: 340,
      child_count: 45,
      staff_count: 12,
    },
  ],
};

const mockEmployeeStaffingData = {
  employees: [],
  months: [],
};

const mockSections = {
  data: [],
  total: 0,
};

describe('StaffingPrintPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getStaffingHours as jest.Mock).mockResolvedValue(mockStaffingData);
    (apiClient.getEmployeeStaffingHours as jest.Mock).mockResolvedValue(mockEmployeeStaffingData);
    (apiClient.getSections as jest.Mock).mockResolvedValue(mockSections);
    window.print = jest.fn();
  });

  it('renders page title', () => {
    renderWithProviders(<StaffingPrintPage />);
    expect(screen.getByText('nav.statisticsStaffing')).toBeInTheDocument();
  });

  it('renders organization name', () => {
    renderWithProviders(<StaffingPrintPage />);
    expect(screen.getByText(/Test Kita/)).toBeInTheDocument();
  });

  it('renders print button', () => {
    renderWithProviders(<StaffingPrintPage />);
    expect(screen.getByText('common.print')).toBeInTheDocument();
  });

  it('calls window.print when print button clicked', async () => {
    renderWithProviders(<StaffingPrintPage />);
    await userEvent.click(screen.getByText('common.print'));
    expect(window.print).toHaveBeenCalled();
  });

  it('renders staffing hours grid section', async () => {
    renderWithProviders(<StaffingPrintPage />);
    await waitFor(() => {
      expect(screen.getByText(/statistics.staffingHoursGrid/)).toBeInTheDocument();
    });
  });

  it('renders employee staffing hours section', async () => {
    renderWithProviders(<StaffingPrintPage />);
    await waitFor(() => {
      expect(screen.getByText(/statistics.employeeStaffingHoursGrid/)).toBeInTheDocument();
    });
  });

  it('has no sidebar or header elements', () => {
    const { container } = renderWithProviders(<StaffingPrintPage />);
    expect(container.querySelector('[class*="sidebar"]')).toBeNull();
    expect(container.querySelector('header')).toBeNull();
  });
});
