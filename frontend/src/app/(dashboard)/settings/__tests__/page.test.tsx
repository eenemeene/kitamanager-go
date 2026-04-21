import { screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import SettingsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    changePassword: jest.fn(),
  },
}));

const toastMock = jest.fn();
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: toastMock }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

beforeEach(() => jest.clearAllMocks());

describe('SettingsPage', () => {
  it('renders the page title and the change-password card', () => {
    renderWithProviders(<SettingsPage />);
    expect(screen.getByText('nav.settings')).toBeInTheDocument();
    expect(screen.getByText('settings.password.title')).toBeInTheDocument();
  });

  it('completes a password change end-to-end', async () => {
    (apiClient.changePassword as jest.Mock).mockResolvedValue({ expires_in: 3600 });
    const u = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await u.type(screen.getByLabelText('settings.password.currentLabel'), 'old-password');
    await u.type(screen.getByLabelText('settings.password.newLabel'), 'new-password-8chars');
    await u.type(screen.getByLabelText('settings.password.confirmLabel'), 'new-password-8chars');
    fireEvent.click(screen.getByRole('button', { name: 'settings.password.submit' }));

    await waitFor(() => {
      expect(apiClient.changePassword).toHaveBeenCalledWith('old-password', 'new-password-8chars');
    });
    expect(toastMock).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'settings.password.successToast' })
    );
  });
});
