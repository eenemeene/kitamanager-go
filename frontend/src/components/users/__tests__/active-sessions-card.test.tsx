import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ActiveSessionsCard } from '../active-sessions-card';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSessions: jest.fn(),
    revokeSession: jest.fn(),
  },
}));

const toastMock = jest.fn();
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: toastMock }),
}));

// translate() identity so assertions read the i18n keys literally.
// `interpolated values come through as a stringified form of the params`
// — good enough for these tests since we never assert on the full text.
jest.mock('next-intl', () => ({
  useTranslations: (ns?: string) => (key: string, values?: Record<string, unknown>) =>
    `${ns ?? ''}${ns ? '.' : ''}${key}${values ? `:${JSON.stringify(values)}` : ''}`,
  useLocale: () => 'en',
}));

function session(
  overrides: Partial<{
    id: string;
    current: boolean;
    user_agent: string;
    ip: string;
    created_at: string;
    expires_at: string;
  }> = {}
) {
  return {
    id: overrides.id ?? 'sess-abc',
    created_at: overrides.created_at ?? new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    expires_at: overrides.expires_at ?? new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
    created_ip: overrides.ip ?? '203.0.113.42',
    created_user_agent: overrides.user_agent ?? 'Chrome 134 / macOS',
    current: overrides.current ?? false,
  };
}

beforeEach(() => {
  jest.clearAllMocks();
});

describe('ActiveSessionsCard', () => {
  it('shows a loading state while the query resolves', async () => {
    // Resolve slowly so the test can observe the loading state.
    (apiClient.getSessions as jest.Mock).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(<ActiveSessionsCard />);

    expect(screen.getByRole('status')).toHaveTextContent('loading');
  });

  it('shows the empty-state copy when the user has no sessions', async () => {
    (apiClient.getSessions as jest.Mock).mockResolvedValue({ sessions: [] });

    renderWithProviders(<ActiveSessionsCard />);

    await waitFor(() => {
      expect(screen.getByText(/empty/i)).toBeInTheDocument();
    });
  });

  it('shows a generic load-error message if the query fails', async () => {
    (apiClient.getSessions as jest.Mock).mockRejectedValue(new Error('boom'));

    renderWithProviders(<ActiveSessionsCard />);

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('loadError');
    });
  });

  it('marks the current session with the "this device" badge and hides its revoke button', async () => {
    (apiClient.getSessions as jest.Mock).mockResolvedValue({
      sessions: [
        session({ id: 'current', current: true, user_agent: 'Current Device' }),
        session({ id: 'other', current: false, user_agent: 'Other Device' }),
      ],
    });

    renderWithProviders(<ActiveSessionsCard />);

    await waitFor(() => {
      expect(screen.getByText(/Current Device/i)).toBeInTheDocument();
    });

    // Exactly one "current" badge — for the current session.
    const badges = screen.getAllByLabelText(/currentBadge/);
    expect(badges).toHaveLength(1);

    // Exactly one Revoke button — for the non-current session. The current
    // session must NOT have one.
    const revokeButtons = screen.getAllByRole('button', { name: /revoke/i });
    expect(revokeButtons).toHaveLength(1);
  });

  it('opens a confirmation dialog before calling the API, and does not call until confirmed', async () => {
    (apiClient.getSessions as jest.Mock).mockResolvedValue({
      sessions: [session({ id: 'target', current: false })],
    });
    const u = userEvent.setup();

    renderWithProviders(<ActiveSessionsCard />);

    await waitFor(() => screen.getByRole('button', { name: /revoke/i }));
    // Click the row-level Revoke → opens dialog.
    await u.click(screen.getByRole('button', { name: /revoke/i }));

    // Dialog visible with a Cancel + Confirm button.
    const dialog = await screen.findByRole('alertdialog');
    expect(dialog).toBeInTheDocument();

    // No API call yet.
    expect(apiClient.revokeSession).not.toHaveBeenCalled();

    // Cancel closes the dialog and still makes no API call.
    await u.click(screen.getByRole('button', { name: /cancel/i }));
    expect(apiClient.revokeSession).not.toHaveBeenCalled();
  });

  it('calls revokeSession with the row id when confirmed, and fires the success toast', async () => {
    (apiClient.getSessions as jest.Mock).mockResolvedValue({
      sessions: [session({ id: 'sess-to-revoke', current: false })],
    });
    (apiClient.revokeSession as jest.Mock).mockResolvedValue(undefined);
    const u = userEvent.setup();

    renderWithProviders(<ActiveSessionsCard />);

    await waitFor(() => screen.getByRole('button', { name: /revoke/i }));
    await u.click(screen.getByRole('button', { name: /revoke/i }));

    // Confirm inside the dialog.
    const confirmButton = (await screen.findAllByRole('button', { name: /revoke/i })).find((el) =>
      el.closest('[role="alertdialog"]')
    );
    expect(confirmButton).toBeDefined();
    await u.click(confirmButton!);

    await waitFor(() => {
      expect(apiClient.revokeSession).toHaveBeenCalledWith('sess-to-revoke');
    });
    await waitFor(() => {
      expect(toastMock).toHaveBeenCalledWith(
        expect.objectContaining({ title: expect.stringContaining('revokeSuccessToast') })
      );
    });
  });

  it('fires a generic error toast on API failure — does not surface backend details', async () => {
    (apiClient.getSessions as jest.Mock).mockResolvedValue({
      sessions: [session({ id: 'sess-err', current: false })],
    });
    (apiClient.revokeSession as jest.Mock).mockRejectedValue({
      response: { status: 404, data: { message: 'session not found' } },
    });
    const u = userEvent.setup();

    renderWithProviders(<ActiveSessionsCard />);

    await waitFor(() => screen.getByRole('button', { name: /revoke/i }));
    await u.click(screen.getByRole('button', { name: /revoke/i }));
    const confirmButton = (await screen.findAllByRole('button', { name: /revoke/i })).find((el) =>
      el.closest('[role="alertdialog"]')
    );
    await u.click(confirmButton!);

    await waitFor(() => {
      expect(toastMock).toHaveBeenCalledWith(
        expect.objectContaining({
          title: expect.stringContaining('revokeErrorGeneric'),
          variant: 'destructive',
        })
      );
    });
    // The raw backend message must NOT surface.
    const flat = JSON.stringify(toastMock.mock.calls);
    expect(flat).not.toContain('session not found');
  });
});
