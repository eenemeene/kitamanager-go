import { screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ChangePasswordCard } from '../change-password-card';
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

beforeEach(() => {
  jest.clearAllMocks();
});

describe('ChangePasswordCard', () => {
  it('renders three password inputs with correct type and autocomplete', () => {
    renderWithProviders(<ChangePasswordCard />);

    const current = screen.getByLabelText('settings.password.currentLabel') as HTMLInputElement;
    const next = screen.getByLabelText('settings.password.newLabel') as HTMLInputElement;
    const confirm = screen.getByLabelText('settings.password.confirmLabel') as HTMLInputElement;

    expect(current.type).toBe('password');
    expect(next.type).toBe('password');
    expect(confirm.type).toBe('password');

    expect(current.autocomplete).toBe('current-password');
    expect(next.autocomplete).toBe('new-password');
    expect(confirm.autocomplete).toBe('new-password');
  });

  it('disables submit while form is invalid (empty state)', () => {
    renderWithProviders(<ChangePasswordCard />);
    expect(screen.getByRole('button', { name: 'settings.password.submit' })).toBeDisabled();
  });

  it('rejects a new password shorter than 8 characters', async () => {
    const u = userEvent.setup();
    renderWithProviders(<ChangePasswordCard />);

    await u.type(screen.getByLabelText('settings.password.currentLabel'), 'old-pass');
    await u.type(screen.getByLabelText('settings.password.newLabel'), 'short');
    await u.type(screen.getByLabelText('settings.password.confirmLabel'), 'short');

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'settings.password.submit' })).toBeDisabled();
    });
    expect(apiClient.changePassword).not.toHaveBeenCalled();
  });

  it('rejects submission when confirm does not match new password', async () => {
    const u = userEvent.setup();
    renderWithProviders(<ChangePasswordCard />);

    await u.type(screen.getByLabelText('settings.password.currentLabel'), 'old-pass');
    await u.type(screen.getByLabelText('settings.password.newLabel'), 'new-password-x');
    await u.type(screen.getByLabelText('settings.password.confirmLabel'), 'new-password-y');

    fireEvent.click(screen.getByRole('button', { name: 'settings.password.submit' }));

    await waitFor(() => {
      const errors = screen.getAllByRole('alert');
      expect(errors.some((e) => e.textContent?.includes('mismatch'))).toBe(true);
    });
    expect(apiClient.changePassword).not.toHaveBeenCalled();
  });

  it('rejects submission when new password equals current password', async () => {
    const u = userEvent.setup();
    renderWithProviders(<ChangePasswordCard />);

    await u.type(screen.getByLabelText('settings.password.currentLabel'), 'same-password');
    await u.type(screen.getByLabelText('settings.password.newLabel'), 'same-password');
    await u.type(screen.getByLabelText('settings.password.confirmLabel'), 'same-password');

    fireEvent.click(screen.getByRole('button', { name: 'settings.password.submit' }));

    await waitFor(() => {
      const errors = screen.getAllByRole('alert');
      expect(errors.some((e) => e.textContent?.includes('sameAsCurrent'))).toBe(true);
    });
    expect(apiClient.changePassword).not.toHaveBeenCalled();
  });

  it('submits the mutation with the right arguments on a valid form', async () => {
    (apiClient.changePassword as jest.Mock).mockResolvedValue({ expires_in: 3600 });
    const u = userEvent.setup();
    renderWithProviders(<ChangePasswordCard />);

    await u.type(screen.getByLabelText('settings.password.currentLabel'), 'old-password');
    await u.type(screen.getByLabelText('settings.password.newLabel'), 'new-password-8chars');
    await u.type(screen.getByLabelText('settings.password.confirmLabel'), 'new-password-8chars');

    fireEvent.click(screen.getByRole('button', { name: 'settings.password.submit' }));

    await waitFor(() => {
      expect(apiClient.changePassword).toHaveBeenCalledWith('old-password', 'new-password-8chars');
    });
  });

  it('shows a generic error toast on backend failure and does NOT echo the backend message', async () => {
    (apiClient.changePassword as jest.Mock).mockRejectedValue({
      response: {
        data: { code: 'unauthorized', message: 'current password is incorrect' },
        status: 401,
      },
    });
    const u = userEvent.setup();
    renderWithProviders(<ChangePasswordCard />);

    await u.type(screen.getByLabelText('settings.password.currentLabel'), 'wrong');
    await u.type(screen.getByLabelText('settings.password.newLabel'), 'new-password-8chars');
    await u.type(screen.getByLabelText('settings.password.confirmLabel'), 'new-password-8chars');
    fireEvent.click(screen.getByRole('button', { name: 'settings.password.submit' }));

    await waitFor(() => {
      expect(toastMock).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'settings.password.errorGeneric',
          variant: 'destructive',
        })
      );
    });
    // Critical: the raw backend message must NOT surface.
    const calls = toastMock.mock.calls.flat();
    for (const call of calls) {
      expect(JSON.stringify(call)).not.toContain('current password is incorrect');
    }
  });

  it('clears the form and shows a success toast after successful change', async () => {
    (apiClient.changePassword as jest.Mock).mockResolvedValue({ expires_in: 3600 });
    const u = userEvent.setup();
    renderWithProviders(<ChangePasswordCard />);

    const current = screen.getByLabelText('settings.password.currentLabel') as HTMLInputElement;
    const next = screen.getByLabelText('settings.password.newLabel') as HTMLInputElement;
    const confirm = screen.getByLabelText('settings.password.confirmLabel') as HTMLInputElement;

    await u.type(current, 'old-password');
    await u.type(next, 'new-password-8chars');
    await u.type(confirm, 'new-password-8chars');
    fireEvent.click(screen.getByRole('button', { name: 'settings.password.submit' }));

    await waitFor(() => {
      expect(toastMock).toHaveBeenCalledWith(
        expect.objectContaining({ title: 'settings.password.successToast' })
      );
    });

    await waitFor(() => {
      expect(current.value).toBe('');
      expect(next.value).toBe('');
      expect(confirm.value).toBe('');
    });
  });

  it('marks fields aria-invalid when they have errors', async () => {
    const u = userEvent.setup();
    renderWithProviders(<ChangePasswordCard />);

    const next = screen.getByLabelText('settings.password.newLabel') as HTMLInputElement;
    await u.type(next, 'short');

    await waitFor(() => {
      expect(next.getAttribute('aria-invalid')).toBe('true');
    });
  });
});
