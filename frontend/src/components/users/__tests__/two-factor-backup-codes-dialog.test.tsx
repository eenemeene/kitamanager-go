import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { TwoFactorBackupCodesDialog } from '../two-factor-backup-codes-dialog';
import { renderWithProviders } from '@/test-utils';
import { copyToClipboard } from '@/lib/utils/clipboard';
import { downloadAsText } from '@/lib/utils/download';

jest.mock('@/lib/utils/clipboard', () => ({
  copyToClipboard: jest.fn(),
}));
jest.mock('@/lib/utils/download', () => ({
  downloadAsText: jest.fn(),
}));

const toastMock = jest.fn();
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: toastMock }),
}));

jest.mock('next-intl', () => ({
  useTranslations: (ns?: string) => (key: string) => `${ns ?? ''}${ns ? '.' : ''}${key}`,
  useLocale: () => 'en',
}));

const payload = {
  factor_id: 99,
  codes: ['aaaa-bbbb', 'cccc-dddd', 'eeee-ffff'],
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe('TwoFactorBackupCodesDialog — closed state', () => {
  it('renders nothing when payload is null', () => {
    renderWithProviders(<TwoFactorBackupCodesDialog payload={null} onClose={jest.fn()} />);
    expect(screen.queryByTestId('backup-codes-list')).not.toBeInTheDocument();
  });
});

describe('TwoFactorBackupCodesDialog — open with codes', () => {
  it('renders all codes', () => {
    renderWithProviders(<TwoFactorBackupCodesDialog payload={payload} onClose={jest.fn()} />);
    expect(screen.getByTestId('backup-code-0')).toHaveTextContent('aaaa-bbbb');
    expect(screen.getByTestId('backup-code-1')).toHaveTextContent('cccc-dddd');
    expect(screen.getByTestId('backup-code-2')).toHaveTextContent('eeee-ffff');
  });

  it('Done is disabled until acknowledge checkbox is ticked', async () => {
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorBackupCodesDialog payload={payload} onClose={jest.fn()} />);
    const done = screen.getByRole('button', {
      name: 'settings.twoFactor.backupCodesDialog.confirm',
    });
    expect(done).toBeDisabled();
    await u.click(screen.getByLabelText('settings.twoFactor.backupCodesDialog.acknowledge'));
    expect(done).toBeEnabled();
  });

  it('Done calls onClose only after acknowledging', async () => {
    const u = userEvent.setup();
    const onClose = jest.fn();
    renderWithProviders(<TwoFactorBackupCodesDialog payload={payload} onClose={onClose} />);
    await u.click(screen.getByLabelText('settings.twoFactor.backupCodesDialog.acknowledge'));
    await u.click(
      screen.getByRole('button', { name: 'settings.twoFactor.backupCodesDialog.confirm' })
    );
    expect(onClose).toHaveBeenCalled();
  });

  it('Copy button invokes copyToClipboard with newline-joined codes + shows success toast', async () => {
    (copyToClipboard as jest.Mock).mockResolvedValue(true);
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorBackupCodesDialog payload={payload} onClose={jest.fn()} />);
    await u.click(
      screen.getByRole('button', { name: /settings\.twoFactor\.backupCodesDialog\.copy/ })
    );
    await waitFor(() => {
      expect(copyToClipboard).toHaveBeenCalledWith('aaaa-bbbb\ncccc-dddd\neeee-ffff');
    });
    expect(toastMock).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'settings.twoFactor.backupCodesDialog.copied' })
    );
  });

  it('Copy shows destructive toast if clipboard fails', async () => {
    (copyToClipboard as jest.Mock).mockResolvedValue(false);
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorBackupCodesDialog payload={payload} onClose={jest.fn()} />);
    await u.click(
      screen.getByRole('button', { name: /settings\.twoFactor\.backupCodesDialog\.copy/ })
    );
    await waitFor(() => {
      expect(toastMock).toHaveBeenCalledWith(expect.objectContaining({ variant: 'destructive' }));
    });
  });

  it('Download invokes downloadAsText with newline-joined codes + filename', async () => {
    const u = userEvent.setup();
    renderWithProviders(<TwoFactorBackupCodesDialog payload={payload} onClose={jest.fn()} />);
    await u.click(
      screen.getByRole('button', { name: /settings\.twoFactor\.backupCodesDialog\.download/ })
    );
    expect(downloadAsText).toHaveBeenCalledWith(
      'aaaa-bbbb\ncccc-dddd\neeee-ffff',
      'settings.twoFactor.backupCodesDialog.filename'
    );
  });
});

describe('TwoFactorBackupCodesDialog — reset on new payload', () => {
  it('a fresh payload resets the acknowledge checkbox (regenerate-after-save flow)', async () => {
    const u = userEvent.setup();
    const { rerender } = renderWithProviders(
      <TwoFactorBackupCodesDialog payload={payload} onClose={jest.fn()} />
    );
    await u.click(screen.getByLabelText('settings.twoFactor.backupCodesDialog.acknowledge'));
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.backupCodesDialog.confirm' })
    ).toBeEnabled();

    // New payload arrives (different codes → different key).
    rerender(
      <TwoFactorBackupCodesDialog
        payload={{ factor_id: 99, codes: ['new1-aaaa', 'new2-bbbb'] }}
        onClose={jest.fn()}
      />
    );
    expect(
      screen.getByRole('button', { name: 'settings.twoFactor.backupCodesDialog.confirm' })
    ).toBeDisabled();
  });
});
