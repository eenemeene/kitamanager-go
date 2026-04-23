import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { AxiosError } from 'axios';

import { TwoFactorEnrolDialog } from '../two-factor-enrol-dialog';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    enrolTotp: jest.fn(),
    activateFactor: jest.fn(),
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

jest.mock('qrcode.react', () => ({
  QRCodeSVG: ({ value }: { value: string }) => <div data-testid="qrcode" data-value={value} />,
}));

function axiosError(status: number): Error {
  const err = new Error(`http ${status}`) as unknown as AxiosError;
  (err as unknown as { response: { status: number } }).response = { status };
  return err as unknown as Error;
}

function seededFactor() {
  return {
    id: 100,
    type: 'totp' as const,
    created_at: '2026-01-01T00:00:00Z',
    activated: false,
    enrollment: {
      secret: 'JBSWY3DPEHPK3PXP',
      otpauth_uri: 'otpauth://totp/KitaManager:u@example.com?secret=JBSWY3DPEHPK3PXP',
    },
  };
}

beforeEach(() => {
  jest.clearAllMocks();
});

describe('TwoFactorEnrolDialog — step: password', () => {
  it('renders the password input + continue disabled until valid', () => {
    const onComplete = jest.fn();
    renderWithProviders(
      <TwoFactorEnrolDialog open onOpenChange={() => {}} onComplete={onComplete} />
    );
    const cont = screen.getByRole('button', { name: 'settings.twoFactor.enrolDialog.continue' });
    expect(cont).toBeDisabled();
  });

  it('enables continue after password is typed + transitions to scan step on success', async () => {
    const u = userEvent.setup();
    (apiClient.enrolTotp as jest.Mock).mockResolvedValue(seededFactor());
    renderWithProviders(
      <TwoFactorEnrolDialog open onOpenChange={() => {}} onComplete={jest.fn()} />
    );
    await u.type(screen.getByLabelText('settings.twoFactor.enrolDialog.passwordLabel'), 'mypass');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.enrolDialog.continue' }));

    await waitFor(() => {
      expect(screen.getByTestId('qrcode')).toBeInTheDocument();
    });
    expect(apiClient.enrolTotp).toHaveBeenCalledWith('mypass', undefined);
  });

  it('shows wrongPassword inline when enrol returns 401 + clears the input', async () => {
    const u = userEvent.setup();
    (apiClient.enrolTotp as jest.Mock).mockRejectedValue(axiosError(401));
    renderWithProviders(
      <TwoFactorEnrolDialog open onOpenChange={() => {}} onComplete={jest.fn()} />
    );
    await u.type(screen.getByLabelText('settings.twoFactor.enrolDialog.passwordLabel'), 'bad');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.enrolDialog.continue' }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.enrolDialog.wrongPassword'
      );
    });
    // Password field cleared after failure.
    expect(
      (screen.getByLabelText('settings.twoFactor.enrolDialog.passwordLabel') as HTMLInputElement)
        .value
    ).toBe('');
  });

  it('cancel calls onOpenChange(false)', async () => {
    const u = userEvent.setup();
    const onOpenChange = jest.fn();
    renderWithProviders(
      <TwoFactorEnrolDialog open onOpenChange={onOpenChange} onComplete={jest.fn()} />
    );
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.enrolDialog.cancel' }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});

describe('TwoFactorEnrolDialog — step: scan', () => {
  async function renderAtScanStep(onComplete = jest.fn(), onOpenChange = jest.fn()) {
    (apiClient.enrolTotp as jest.Mock).mockResolvedValue(seededFactor());
    const result = renderWithProviders(
      <TwoFactorEnrolDialog open onOpenChange={onOpenChange} onComplete={onComplete} />
    );
    const u = userEvent.setup();
    await u.type(screen.getByLabelText('settings.twoFactor.enrolDialog.passwordLabel'), 'mypass');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.enrolDialog.continue' }));
    await waitFor(() => expect(screen.getByTestId('qrcode')).toBeInTheDocument());
    return { ...result, u };
  }

  it('renders the QR code with the otpauth URI', async () => {
    await renderAtScanStep();
    expect(screen.getByTestId('qrcode')).toHaveAttribute(
      'data-value',
      'otpauth://totp/KitaManager:u@example.com?secret=JBSWY3DPEHPK3PXP'
    );
  });

  it('renders the fallback-secret hint', async () => {
    await renderAtScanStep();
    expect(screen.getByText('JBSWY3DPEHPK3PXP')).toBeInTheDocument();
  });

  it('calls onComplete with backup codes payload on successful activation', async () => {
    const onComplete = jest.fn();
    (apiClient.activateFactor as jest.Mock).mockResolvedValue({
      activated: true,
      backup_codes: { factor_id: 101, codes: ['aaaa-bbbb', 'cccc-dddd'] },
    });
    const { u } = await renderAtScanStep(onComplete);
    await u.type(screen.getByLabelText('settings.twoFactor.enrolDialog.codeLabel'), '123456');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.enrolDialog.activate' }));
    await waitFor(() =>
      expect(onComplete).toHaveBeenCalledWith({
        factor_id: 101,
        codes: ['aaaa-bbbb', 'cccc-dddd'],
      })
    );
  });

  it('shows wrongCode inline + clears input on 401', async () => {
    (apiClient.activateFactor as jest.Mock).mockRejectedValue(axiosError(401));
    const { u } = await renderAtScanStep();
    const codeInput = screen.getByLabelText('settings.twoFactor.enrolDialog.codeLabel');
    await u.type(codeInput, '000000');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.enrolDialog.activate' }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.enrolDialog.wrongCode'
      );
    });
    // Input cleared.
    expect((codeInput as HTMLInputElement).value).toBe('');
  });

  it('closes dialog + toasts tooMany on 429', async () => {
    const onOpenChange = jest.fn();
    (apiClient.activateFactor as jest.Mock).mockRejectedValue(axiosError(429));
    const { u } = await renderAtScanStep(jest.fn(), onOpenChange);
    await u.type(screen.getByLabelText('settings.twoFactor.enrolDialog.codeLabel'), '000000');
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.enrolDialog.activate' }));
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(toastMock).toHaveBeenCalledWith(expect.objectContaining({ variant: 'destructive' }));
  });
});
