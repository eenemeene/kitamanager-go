import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { AxiosError } from 'axios';

import { TwoFactorDisableDialog } from '../two-factor-disable-dialog';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    deleteFactor: jest.fn(),
  },
}));

const toastMock = jest.fn();
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: toastMock }),
}));

jest.mock('next-intl', () => ({
  useTranslations: (ns?: string) => (key: string) => `${ns ?? ''}${ns ? '.' : ''}${key}`,
  useLocale: () => 'en',
}));

function axiosError(status: number): Error {
  const err = new Error(`http ${status}`) as unknown as AxiosError;
  (err as unknown as { response: { status: number } }).response = { status };
  return err as unknown as Error;
}

beforeEach(() => {
  jest.clearAllMocks();
});

describe('TwoFactorDisableDialog', () => {
  it('submit disabled when password or code is empty', async () => {
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorDisableDialog open onOpenChange={() => {}} factorId={42} />);
    const submit = screen.getByRole('button', {
      name: 'settings.twoFactor.disableDialog.submit',
    });
    expect(submit).toBeDisabled();

    await u.type(screen.getByLabelText('settings.twoFactor.disableDialog.passwordLabel'), 'pw');
    expect(submit).toBeDisabled();

    await u.type(screen.getByLabelText('settings.twoFactor.disableDialog.codeLabel'), '123456');
    expect(submit).toBeEnabled();
  });

  it('posts password + code on submit + closes dialog on 204', async () => {
    const onOpenChange = jest.fn();
    (apiClient.deleteFactor as jest.Mock).mockResolvedValue(undefined);
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorDisableDialog open onOpenChange={onOpenChange} factorId={42} />);
    await u.type(screen.getByLabelText('settings.twoFactor.disableDialog.passwordLabel'), 'pw');
    await u.type(screen.getByLabelText('settings.twoFactor.disableDialog.codeLabel'), '123456');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.disableDialog.submit' }));
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(apiClient.deleteFactor).toHaveBeenCalledWith(42, 'pw', '123456');
  });

  it('shows wrongPasswordOrCode on 401 + clears both inputs', async () => {
    (apiClient.deleteFactor as jest.Mock).mockRejectedValue(axiosError(401));
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorDisableDialog open onOpenChange={() => {}} factorId={42} />);
    const pw = screen.getByLabelText('settings.twoFactor.disableDialog.passwordLabel');
    const code = screen.getByLabelText('settings.twoFactor.disableDialog.codeLabel');
    await u.type(pw, 'pw');
    await u.type(code, '000000');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.disableDialog.submit' }));
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.disableDialog.wrongPasswordOrCode'
      );
    });
    expect((pw as HTMLInputElement).value).toBe('');
    expect((code as HTMLInputElement).value).toBe('');
  });

  it('shows wrongPasswordOrCode on 400 (backend rejected missing-code last-primary)', async () => {
    (apiClient.deleteFactor as jest.Mock).mockRejectedValue(axiosError(400));
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorDisableDialog open onOpenChange={() => {}} factorId={42} />);
    await u.type(screen.getByLabelText('settings.twoFactor.disableDialog.passwordLabel'), 'pw');
    await u.type(screen.getByLabelText('settings.twoFactor.disableDialog.codeLabel'), 'code');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.disableDialog.submit' }));
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.disableDialog.wrongPasswordOrCode'
      );
    });
  });

  it('cancel closes dialog', async () => {
    const onOpenChange = jest.fn();
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorDisableDialog open onOpenChange={onOpenChange} factorId={42} />);
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.disableDialog.cancel' }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
