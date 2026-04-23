import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { AxiosError } from 'axios';

import { MfaVerifyForm } from '../mfa-verify-form';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';
import type { LoginFactorDescriptor } from '@/lib/api/types';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    verifyMfa: jest.fn(),
  },
}));

const hydrateMock = jest.fn().mockResolvedValue(undefined);
jest.mock('@/stores/auth-store', () => ({
  useAuthStore: (selector: (s: unknown) => unknown) => selector({ hydrateAfterAuth: hydrateMock }),
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

const factorsOne: LoginFactorDescriptor[] = [{ id: 42, type: 'totp', label: 'iPhone' }];
const factorsTwo: LoginFactorDescriptor[] = [
  { id: 42, type: 'totp', label: 'iPhone' },
  { id: 43, type: 'backup_codes' },
];

beforeEach(() => {
  jest.clearAllMocks();
});

describe('MfaVerifyForm — initial render', () => {
  it('renders with submit disabled until a code is typed', () => {
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="tok"
        factors={factorsOne}
        onRestart={jest.fn()}
        onSuccess={jest.fn()}
      />
    );
    expect(screen.getByRole('button', { name: 'auth.mfa.submit' })).toBeDisabled();
  });

  it('hides factor picker when only one factor is present', () => {
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="tok"
        factors={factorsOne}
        onRestart={jest.fn()}
        onSuccess={jest.fn()}
      />
    );
    expect(screen.queryByLabelText('auth.mfa.factorPickerLabel')).not.toBeInTheDocument();
  });

  it('shows factor picker when multiple factors are present', () => {
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="tok"
        factors={factorsTwo}
        onRestart={jest.fn()}
        onSuccess={jest.fn()}
      />
    );
    expect(screen.getByLabelText('auth.mfa.factorPickerLabel')).toBeInTheDocument();
  });
});

describe('MfaVerifyForm — submit: success', () => {
  it('posts pending_token + factor_id + code and calls onSuccess after hydration', async () => {
    (apiClient.verifyMfa as jest.Mock).mockResolvedValue({
      status: 'authenticated',
      expires_in: 604800,
    });
    const onSuccess = jest.fn();
    const u = userEvent.setup();
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="the-token"
        factors={factorsOne}
        onRestart={jest.fn()}
        onSuccess={onSuccess}
      />
    );
    await u.type(screen.getByLabelText('auth.mfa.codeLabel'), '123456');
    await u.click(screen.getByRole('button', { name: 'auth.mfa.submit' }));

    await waitFor(() => {
      expect(apiClient.verifyMfa).toHaveBeenCalledWith({
        pending_token: 'the-token',
        factor_id: 42,
        code: '123456',
      });
    });
    expect(hydrateMock).toHaveBeenCalled();
    expect(onSuccess).toHaveBeenCalled();
  });
});

describe('MfaVerifyForm — submit: 401 (wrong code)', () => {
  it('stays on the form, shows wrongCode alert, clears the input', async () => {
    (apiClient.verifyMfa as jest.Mock).mockRejectedValue(axiosError(401));
    const onSuccess = jest.fn();
    const onRestart = jest.fn();
    const u = userEvent.setup();
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="tok"
        factors={factorsOne}
        onRestart={onRestart}
        onSuccess={onSuccess}
      />
    );
    const codeInput = screen.getByLabelText('auth.mfa.codeLabel');
    await u.type(codeInput, '000000');
    await u.click(screen.getByRole('button', { name: 'auth.mfa.submit' }));
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('auth.mfa.wrongCode');
    });
    expect((codeInput as HTMLInputElement).value).toBe('');
    expect(onRestart).not.toHaveBeenCalled();
    expect(onSuccess).not.toHaveBeenCalled();
  });
});

describe('MfaVerifyForm — submit: 429 (rate limit)', () => {
  it('calls onRestart("too_many") so the page reverts to password step', async () => {
    (apiClient.verifyMfa as jest.Mock).mockRejectedValue(axiosError(429));
    const onRestart = jest.fn();
    const u = userEvent.setup();
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="tok"
        factors={factorsOne}
        onRestart={onRestart}
        onSuccess={jest.fn()}
      />
    );
    await u.type(screen.getByLabelText('auth.mfa.codeLabel'), '000000');
    await u.click(screen.getByRole('button', { name: 'auth.mfa.submit' }));
    await waitFor(() => expect(onRestart).toHaveBeenCalledWith('too_many'));
  });
});

describe('MfaVerifyForm — back button', () => {
  it('calls onRestart("user")', async () => {
    const onRestart = jest.fn();
    const u = userEvent.setup();
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="tok"
        factors={factorsOne}
        onRestart={onRestart}
        onSuccess={jest.fn()}
      />
    );
    await u.click(screen.getByRole('button', { name: 'auth.mfa.back' }));
    expect(onRestart).toHaveBeenCalledWith('user');
  });
});

describe('MfaVerifyForm — recovery path', () => {
  it('after a wrong-code 401 a second submission with the correct code succeeds', async () => {
    (apiClient.verifyMfa as jest.Mock)
      .mockRejectedValueOnce(axiosError(401))
      .mockResolvedValueOnce({ status: 'authenticated', expires_in: 1 });
    const onSuccess = jest.fn();
    const u = userEvent.setup();
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="tok"
        factors={factorsOne}
        onRestart={jest.fn()}
        onSuccess={onSuccess}
      />
    );
    const codeInput = screen.getByLabelText('auth.mfa.codeLabel');
    await u.type(codeInput, '000000');
    await u.click(screen.getByRole('button', { name: 'auth.mfa.submit' }));
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('auth.mfa.wrongCode'));
    // Input cleared; type the real code.
    await u.type(codeInput, '123456');
    await u.click(screen.getByRole('button', { name: 'auth.mfa.submit' }));
    await waitFor(() => expect(onSuccess).toHaveBeenCalled());
  });
});

describe('MfaVerifyForm — multi-factor selection', () => {
  it('defaults to the first factor in the list (TOTP appears first server-side)', async () => {
    (apiClient.verifyMfa as jest.Mock).mockResolvedValue({
      status: 'authenticated',
      expires_in: 1,
    });
    const u = userEvent.setup();
    renderWithProviders(
      <MfaVerifyForm
        pendingToken="tok"
        factors={factorsTwo}
        onRestart={jest.fn()}
        onSuccess={jest.fn()}
      />
    );
    await u.type(screen.getByLabelText('auth.mfa.codeLabel'), '123456');
    await u.click(screen.getByRole('button', { name: 'auth.mfa.submit' }));
    await waitFor(() =>
      // First factor (id=42, TOTP) is used by default. Radix Select's
      // full DOM interaction is brittle under JSDOM so we assert on
      // the default selection here; e2e tests exercise the picker-
      // switch case against a real browser.
      expect(apiClient.verifyMfa).toHaveBeenCalledWith({
        pending_token: 'tok',
        factor_id: 42,
        code: '123456',
      })
    );
  });
});
