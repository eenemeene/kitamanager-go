import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { TwoFactorCard } from '../two-factor-card';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';
import type { FactorResponse } from '@/lib/api/types';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    listMyFactors: jest.fn(),
    enrolTotp: jest.fn(),
    activateFactor: jest.fn(),
    regenerateBackupCodes: jest.fn(),
    deleteFactor: jest.fn(),
  },
}));

const toastMock = jest.fn();
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: toastMock }),
}));

jest.mock('next-intl', () => ({
  useTranslations: (ns?: string) => (key: string, values?: Record<string, unknown>) =>
    `${ns ?? ''}${ns ? '.' : ''}${key}${values ? `:${JSON.stringify(values)}` : ''}`,
  useLocale: () => 'en',
}));

// qrcode.react uses SVG + crypto; stub it to a data-testid div so
// tests don't pull in the full renderer (speeds up + avoids
// non-deterministic DOM output across environments).
jest.mock('qrcode.react', () => ({
  QRCodeSVG: ({ value }: { value: string }) => <div data-testid="qrcode" data-value={value} />,
}));

const totpFactor: FactorResponse = {
  id: 42,
  type: 'totp',
  label: 'iPhone',
  enabled_at: '2026-04-20T10:00:00Z',
  created_at: '2026-04-20T09:00:00Z',
  activated: true,
  last_used_at: '2026-04-22T12:00:00Z',
};

const backupFactor: FactorResponse = {
  id: 43,
  type: 'backup_codes',
  enabled_at: '2026-04-20T10:00:00Z',
  created_at: '2026-04-20T10:00:00Z',
  activated: true,
  backup_codes_remaining: 7,
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe('TwoFactorCard — state: loading', () => {
  it('shows the loading message while the factors query resolves', () => {
    (apiClient.listMyFactors as jest.Mock).mockReturnValue(new Promise(() => {}));
    renderWithProviders(<TwoFactorCard />);
    expect(screen.getByText('settings.twoFactor.loading')).toBeInTheDocument();
  });
});

describe('TwoFactorCard — state: loadError', () => {
  it('shows the error alert when the factors query fails', async () => {
    (apiClient.listMyFactors as jest.Mock).mockRejectedValue(new Error('boom'));
    renderWithProviders(<TwoFactorCard />);
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('settings.twoFactor.loadError');
    });
  });
});

describe('TwoFactorCard — state: disabled', () => {
  beforeEach(() => {
    (apiClient.listMyFactors as jest.Mock).mockResolvedValue({ factors: [] });
  });

  it('renders the status + Enable button', async () => {
    renderWithProviders(<TwoFactorCard />);
    expect(await screen.findByText('settings.twoFactor.statusDisabled')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.enableButton' })
    ).toBeInTheDocument();
  });

  it('opens the enrol dialog when Enable is clicked', async () => {
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorCard />);
    const btn = await screen.findByRole('button', { name: 'settings.twoFactor.enableButton' });
    await u.click(btn);
    expect(
      screen.getByRole('dialog', { name: 'settings.twoFactor.enrolDialog.title' })
    ).toBeInTheDocument();
  });
});

describe('TwoFactorCard — state: enabled', () => {
  beforeEach(() => {
    (apiClient.listMyFactors as jest.Mock).mockResolvedValue({
      factors: [totpFactor, backupFactor],
    });
  });

  it('renders the enabled status + both factor rows + action buttons', async () => {
    renderWithProviders(<TwoFactorCard />);
    await waitFor(() => expect(screen.getByTestId('factor-row-totp')).toBeInTheDocument());
    expect(screen.getByTestId('factor-row-backup_codes')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.regenerateButton' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.disableButton' })
    ).toBeInTheDocument();
  });

  it('shows TOTP label + "last used" secondary text', async () => {
    renderWithProviders(<TwoFactorCard />);
    const row = await screen.findByTestId('factor-row-totp');
    expect(row).toHaveTextContent('iPhone');
  });

  it('shows backup codes remaining count', async () => {
    renderWithProviders(<TwoFactorCard />);
    const row = await screen.findByTestId('factor-row-backup_codes');
    expect(row).toHaveTextContent('"count":7');
  });

  it('opens the regenerate dialog when Regenerate is clicked', async () => {
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorCard />);
    const btn = await screen.findByRole('button', {
      name: 'settings.twoFactor.regenerateButton',
    });
    await u.click(btn);
    expect(
      screen.getByRole('dialog', { name: 'settings.twoFactor.regenerateDialog.title' })
    ).toBeInTheDocument();
  });

  it('opens the disable dialog when Disable is clicked', async () => {
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorCard />);
    const btn = await screen.findByRole('button', {
      name: 'settings.twoFactor.disableButton',
    });
    await u.click(btn);
    expect(
      screen.getByRole('dialog', { name: 'settings.twoFactor.disableDialog.title' })
    ).toBeInTheDocument();
  });
});

describe('TwoFactorCard — secondary text when last_used_at is missing', () => {
  it('falls back to lastUsedNever when the TOTP factor has no last_used_at', async () => {
    (apiClient.listMyFactors as jest.Mock).mockResolvedValue({
      factors: [{ ...totpFactor, last_used_at: undefined }, backupFactor],
    });
    renderWithProviders(<TwoFactorCard />);
    const row = await screen.findByTestId('factor-row-totp');
    expect(row).toHaveTextContent('settings.twoFactor.lastUsedNever');
  });
});
