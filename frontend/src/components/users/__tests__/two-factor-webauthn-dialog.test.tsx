import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { AxiosError } from 'axios';

import { TwoFactorWebAuthnDialog } from '../two-factor-webauthn-dialog';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    enrolWebAuthn: jest.fn(),
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

// webauthn.ts calls atob/btoa and inspects PublicKeyCredential; mock
// the browser-surface helpers so tests don't need jsdom's (absent)
// WebAuthn API.
jest.mock('@/lib/utils/webauthn', () => {
  const actual = jest.requireActual('@/lib/utils/webauthn');
  return {
    ...actual,
    decodeCreationOptions: jest.fn(() => ({
      challenge: new Uint8Array([1]).buffer,
      rp: { id: 'localhost', name: 'KitaManager' },
      user: { id: new Uint8Array([2]).buffer, name: 'u', displayName: 'U' },
      pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
    })),
    encodeRegistrationResponse: jest.fn(() => ({ id: 'cred', rawId: 'cred', type: 'public-key' })),
    isWebAuthnSupported: jest.fn(() => true),
  };
});

import { isWebAuthnSupported } from '@/lib/utils/webauthn';

// Stub navigator.credentials.create for the happy + error paths.
// jsdom doesn't implement WebAuthn, so we inject the CredentialsContainer
// surface we need directly on navigator.
function stubCredentials(impl: () => Promise<unknown>) {
  Object.defineProperty(window.navigator, 'credentials', {
    configurable: true,
    value: { create: jest.fn().mockImplementation(impl), get: jest.fn() },
  });
}

function axiosError(status: number): Error {
  const err = new Error(`http ${status}`) as unknown as AxiosError;
  (err as unknown as { response: { status: number } }).response = { status };
  return err as unknown as Error;
}

function domError(name: string): Error {
  const err = new Error(name);
  (err as unknown as { name: string }).name = name;
  return err;
}

function enrolResponse() {
  return {
    id: 200,
    type: 'webauthn' as const,
    created_at: '2026-04-23T00:00:00Z',
    activated: false,
    enrollment: { creation_options: { publicKey: {} } },
  };
}

beforeEach(() => {
  jest.clearAllMocks();
  (isWebAuthnSupported as jest.Mock).mockReturnValue(true);
});

describe('TwoFactorWebAuthnDialog — unsupported browser', () => {
  it('shows the unsupported message in place of the form', () => {
    (isWebAuthnSupported as jest.Mock).mockReturnValue(false);
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={() => {}} onComplete={jest.fn()} />
    );
    expect(screen.getByRole('alert')).toHaveTextContent(
      'settings.twoFactor.webauthnDialog.unsupported'
    );
    expect(
      screen.queryByRole('button', { name: 'settings.twoFactor.webauthnDialog.continue' })
    ).not.toBeInTheDocument();
  });
});

describe('TwoFactorWebAuthnDialog — password step', () => {
  it('disables Continue until a password is typed', () => {
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={() => {}} onComplete={jest.fn()} />
    );
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.continue' })
    ).toBeDisabled();
  });

  it('runs the full ceremony and emits backup codes on first-primary activation', async () => {
    const u = userEvent.setup();
    (apiClient.enrolWebAuthn as jest.Mock).mockResolvedValue(enrolResponse());
    stubCredentials(async () => ({
      id: 'cred',
      rawId: new Uint8Array([1]).buffer,
      type: 'public-key',
      response: {
        attestationObject: new Uint8Array([2]).buffer,
        clientDataJSON: new Uint8Array([3]).buffer,
        getTransports: () => ['internal'],
      },
      getClientExtensionResults: () => ({}),
    }));
    (apiClient.activateFactor as jest.Mock).mockResolvedValue({
      activated: true,
      backup_codes: { factor_id: 201, codes: ['aa-bb', 'cc-dd'] },
    });
    const onComplete = jest.fn();
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={() => {}} onComplete={onComplete} />
    );
    await u.type(
      screen.getByLabelText('settings.twoFactor.webauthnDialog.passwordLabel'),
      'mypass'
    );
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.continue' })
    );
    await waitFor(() =>
      expect(onComplete).toHaveBeenCalledWith({
        factor_id: 201,
        codes: ['aa-bb', 'cc-dd'],
      })
    );
    expect(apiClient.activateFactor).toHaveBeenCalledWith(200, {
      webauthnResponse: expect.objectContaining({ id: 'cred' }),
    });
  });

  it('emits null payload when activation returns no backup codes (second-plus factor)', async () => {
    const u = userEvent.setup();
    (apiClient.enrolWebAuthn as jest.Mock).mockResolvedValue(enrolResponse());
    stubCredentials(async () => ({
      id: 'cred',
      rawId: new Uint8Array([1]).buffer,
      type: 'public-key',
      response: {
        attestationObject: new Uint8Array([2]).buffer,
        clientDataJSON: new Uint8Array([3]).buffer,
      },
      getClientExtensionResults: () => ({}),
    }));
    (apiClient.activateFactor as jest.Mock).mockResolvedValue({ activated: true });
    const onComplete = jest.fn();
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={() => {}} onComplete={onComplete} />
    );
    await u.type(
      screen.getByLabelText('settings.twoFactor.webauthnDialog.passwordLabel'),
      'mypass'
    );
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.continue' })
    );
    await waitFor(() => expect(onComplete).toHaveBeenCalledWith(null));
  });

  it('on 401 step-up: reverts to password step, shows wrongPassword inline, clears input', async () => {
    const u = userEvent.setup();
    (apiClient.enrolWebAuthn as jest.Mock).mockRejectedValue(axiosError(401));
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={() => {}} onComplete={jest.fn()} />
    );
    const pwInput = screen.getByLabelText('settings.twoFactor.webauthnDialog.passwordLabel');
    await u.type(pwInput, 'bad');
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.continue' })
    );
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.webauthnDialog.wrongPassword'
      );
    });
    expect((pwInput as HTMLInputElement).value).toBe('');
  });

  it('on 409 already registered: shows the alreadyRegistered prompt error', async () => {
    const u = userEvent.setup();
    (apiClient.enrolWebAuthn as jest.Mock).mockRejectedValue(axiosError(409));
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={() => {}} onComplete={jest.fn()} />
    );
    await u.type(screen.getByLabelText('settings.twoFactor.webauthnDialog.passwordLabel'), 'pw');
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.continue' })
    );
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.webauthnDialog.alreadyRegistered'
      );
    });
  });

  it('on NotAllowedError from the authenticator: shows userCancelled + retry button', async () => {
    const u = userEvent.setup();
    (apiClient.enrolWebAuthn as jest.Mock).mockResolvedValue(enrolResponse());
    stubCredentials(async () => {
      throw domError('NotAllowedError');
    });
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={() => {}} onComplete={jest.fn()} />
    );
    await u.type(screen.getByLabelText('settings.twoFactor.webauthnDialog.passwordLabel'), 'pw');
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.continue' })
    );
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.webauthnDialog.userCancelled'
      );
    });
    // Retry button only appears after a promptError is set.
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.retry' })
    ).toBeInTheDocument();
  });

  it('on InvalidStateError (duplicate credential): shows alreadyRegistered', async () => {
    const u = userEvent.setup();
    (apiClient.enrolWebAuthn as jest.Mock).mockResolvedValue(enrolResponse());
    stubCredentials(async () => {
      throw domError('InvalidStateError');
    });
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={() => {}} onComplete={jest.fn()} />
    );
    await u.type(screen.getByLabelText('settings.twoFactor.webauthnDialog.passwordLabel'), 'pw');
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.continue' })
    );
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'settings.twoFactor.webauthnDialog.alreadyRegistered'
      );
    });
  });

  it('Cancel button calls onOpenChange(false)', async () => {
    const u = userEvent.setup();
    const onOpenChange = jest.fn();
    renderWithProviders(
      <TwoFactorWebAuthnDialog open onOpenChange={onOpenChange} onComplete={jest.fn()} />
    );
    await u.click(screen.getByRole('button', { name: 'settings.twoFactor.webauthnDialog.cancel' }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
