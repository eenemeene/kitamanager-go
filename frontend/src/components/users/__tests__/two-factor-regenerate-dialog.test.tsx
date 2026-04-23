import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { AxiosError } from 'axios';

import { TwoFactorRegenerateDialog } from '../two-factor-regenerate-dialog';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    regenerateBackupCodes: jest.fn(),
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

describe('TwoFactorRegenerateDialog', () => {
  it('submit is disabled when password is empty', () => {
    renderWithProviders(
      <TwoFactorRegenerateDialog
        open
        onOpenChange={() => {}}
        factorId={43}
        onComplete={jest.fn()}
      />
    );
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.regenerateDialog.submit' })
    ).toBeDisabled();
  });

  it('submit is disabled when factorId is undefined', async () => {
    const u = userEvent.setup();
    renderWithProviders(
      <TwoFactorRegenerateDialog
        open
        onOpenChange={() => {}}
        factorId={undefined}
        onComplete={jest.fn()}
      />
    );
    await u.type(screen.getByLabelText('settings.twoFactor.regenerateDialog.passwordLabel'), 'pw');
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.regenerateDialog.submit' })
    ).toBeDisabled();
  });

  it('successful regenerate calls onComplete with the payload', async () => {
    const onComplete = jest.fn();
    (apiClient.regenerateBackupCodes as jest.Mock).mockResolvedValue({
      factor_id: 43,
      codes: ['x-y', 'a-b'],
    });
    const u = userEvent.setup();
    renderWithProviders(
      <TwoFactorRegenerateDialog
        open
        onOpenChange={() => {}}
        factorId={43}
        onComplete={onComplete}
      />
    );
    await u.type(
      screen.getByLabelText('settings.twoFactor.regenerateDialog.passwordLabel'),
      'correct'
    );
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.regenerateDialog.submit' })
    );
    await waitFor(() =>
      expect(onComplete).toHaveBeenCalledWith({ factor_id: 43, codes: ['x-y', 'a-b'] })
    );
    expect(apiClient.regenerateBackupCodes).toHaveBeenCalledWith(43, 'correct');
  });

  it('shows wrongPassword inline on 401 + clears the input', async () => {
    (apiClient.regenerateBackupCodes as jest.Mock).mockRejectedValue(axiosError(401));
    const u = userEvent.setup();
    renderWithProviders(
      <TwoFactorRegenerateDialog
        open
        onOpenChange={() => {}}
        factorId={43}
        onComplete={jest.fn()}
      />
    );
    const input = screen.getByLabelText('settings.twoFactor.regenerateDialog.passwordLabel');
    await u.type(input, 'bad');
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.regenerateDialog.submit' })
    );
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.regenerateDialog.wrongPassword'
      );
    });
    expect((input as HTMLInputElement).value).toBe('');
  });

  it('cancel calls onOpenChange(false)', async () => {
    const onOpenChange = jest.fn();
    const u = userEvent.setup();
    renderWithProviders(
      <TwoFactorRegenerateDialog
        open
        onOpenChange={onOpenChange}
        factorId={43}
        onComplete={jest.fn()}
      />
    );
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.regenerateDialog.cancel' })
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
