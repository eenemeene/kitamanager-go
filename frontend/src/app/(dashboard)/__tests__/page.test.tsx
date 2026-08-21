import { screen } from '@testing-library/react';
import RootRedirectPage from '../page';
import { useUiStore } from '@/stores/ui-store';
import { renderWithProviders } from '@/test-utils';

const mockReplace = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ replace: mockReplace }),
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/stores/ui-store', () => ({
  useUiStore: jest.fn(),
}));

describe('RootRedirectPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('redirects to selected org dashboard', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [{ id: 1 }, { id: 2 }],
      organizationsLoading: false,
      selectedOrganizationId: 2,
    });

    renderWithProviders(<RootRedirectPage />);

    expect(mockReplace).toHaveBeenCalledWith('/organizations/2/dashboard');
  });

  it('redirects to first org when none selected', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [{ id: 5 }],
      organizationsLoading: false,
      selectedOrganizationId: null,
    });

    renderWithProviders(<RootRedirectPage />);

    expect(mockReplace).toHaveBeenCalledWith('/organizations/5/dashboard');
  });

  it('shows loading skeleton while orgs load', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: true,
      selectedOrganizationId: null,
    });

    renderWithProviders(<RootRedirectPage />);

    expect(mockReplace).not.toHaveBeenCalled();
  });

  it('shows fallback when no orgs exist', () => {
    (useUiStore as unknown as jest.Mock).mockReturnValue({
      organizations: [],
      organizationsLoading: false,
      selectedOrganizationId: null,
    });

    renderWithProviders(<RootRedirectPage />);

    expect(screen.getByText('dashboard.title')).toBeInTheDocument();
    expect(screen.getByText('statistics.selectOrgForStats')).toBeInTheDocument();
    expect(mockReplace).not.toHaveBeenCalled();
  });
});
