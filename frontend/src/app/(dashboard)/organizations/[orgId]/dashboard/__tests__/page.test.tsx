import { screen } from '@testing-library/react';
import DashboardPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string) => key;
    t.has = () => false;
    return t;
  },
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getEmployees: jest.fn(),
    getChildren: jest.fn(),
    getStaffingHours: jest.fn(),
    getStepPromotions: jest.fn(),
    getUpcomingChildren: jest.fn(),
    getChildrenAll: jest.fn(),
    getSections: jest.fn(),
    updateChildContract: jest.fn(),
  },
  getErrorMessage: jest.fn((_e: unknown, f: string) => f),
}));

jest.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({ user: { name: 'Admin' } }),
}));

describe('DashboardPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getEmployees as jest.Mock).mockResolvedValue(createMockPaginatedResponse([]));
    (apiClient.getChildren as jest.Mock).mockResolvedValue(createMockPaginatedResponse([]));
    (apiClient.getStaffingHours as jest.Mock).mockResolvedValue({ data_points: [] });
    (apiClient.getStepPromotions as jest.Mock).mockResolvedValue({
      promotions: [],
      total_monthly_cost_delta: 0,
    });
    (apiClient.getUpcomingChildren as jest.Mock).mockResolvedValue([]);
    (apiClient.getChildrenAll as jest.Mock).mockResolvedValue([]);
    (apiClient.getSections as jest.Mock).mockResolvedValue(createMockPaginatedResponse([]));
  });

  it('renders welcome message', () => {
    renderWithProviders(<DashboardPage />);
    expect(screen.getByText(/welcome/i)).toBeInTheDocument();
  });

  it('renders stat cards', () => {
    renderWithProviders(<DashboardPage />);
    // Loading skeletons should be present
    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });
});
