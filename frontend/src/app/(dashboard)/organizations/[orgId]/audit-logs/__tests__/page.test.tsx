import { screen, waitFor, fireEvent } from '@testing-library/react';
import AuditLogPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getAuditLogs: jest.fn(),
  },
  getErrorMessage: jest.fn((_error, fallback) => fallback),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
  useLocale: () => 'en',
}));

const mockEntries = [
  {
    id: 42,
    timestamp: '2026-04-21T10:00:00Z',
    user_id: 7,
    user_email: 'admin@test.com',
    action: 'child_delete',
    resource_type: 'child',
    resource_id: 99,
    organization_id: 1,
    ip_address: '127.0.0.1',
    details: '{"resource_name":"Anna"}',
    success: true,
  },
  {
    id: 41,
    timestamp: '2026-04-21T09:00:00Z',
    user_id: 7,
    user_email: 'admin@test.com',
    action: 'role_change',
    resource_type: 'user_organization',
    resource_id: 5,
    organization_id: 1,
    ip_address: '127.0.0.1',
    details: '{"old_role":"manager","new_role":"admin"}',
    success: true,
  },
];

describe('AuditLogPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders rows returned by the API and calls getAuditLogs with defaults', async () => {
    (apiClient.getAuditLogs as jest.Mock).mockResolvedValueOnce({
      data: mockEntries,
      total: 2,
      page: 1,
      limit: 20,
    });

    renderWithProviders(<AuditLogPage />);

    // Each row is rendered in both the desktop table and the mobile card list,
    // so the matching node appears twice in jsdom. That's fine — we only care
    // that the action surfaces at all.
    await waitFor(() => {
      expect(screen.getAllByText('child_delete').length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText('role_change').length).toBeGreaterThan(0);
    expect(apiClient.getAuditLogs).toHaveBeenCalledWith(1, {
      page: 1,
      limit: 20,
      action: undefined,
      from: undefined,
      to: undefined,
    });
  });

  it('shows the empty-state copy when the list is empty', async () => {
    (apiClient.getAuditLogs as jest.Mock).mockResolvedValueOnce({
      data: [],
      total: 0,
      page: 1,
      limit: 20,
    });

    renderWithProviders(<AuditLogPage />);

    await waitFor(() => {
      expect(screen.getByText('auditLog.empty')).toBeInTheDocument();
    });
  });

  it('forwards the action filter into the API call', async () => {
    (apiClient.getAuditLogs as jest.Mock).mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
      limit: 20,
    });

    renderWithProviders(<AuditLogPage />);

    await waitFor(() => {
      expect(apiClient.getAuditLogs).toHaveBeenCalled();
    });

    fireEvent.change(screen.getByLabelText('auditLog.action'), {
      target: { value: 'child_delete' },
    });

    await waitFor(() => {
      const lastCall = (apiClient.getAuditLogs as jest.Mock).mock.calls.at(-1);
      expect(lastCall?.[1]).toMatchObject({ action: 'child_delete', page: 1 });
    });
  });

  it('opens the detail dialog when a row is clicked', async () => {
    (apiClient.getAuditLogs as jest.Mock).mockResolvedValueOnce({
      data: mockEntries,
      total: 2,
      page: 1,
      limit: 20,
    });

    renderWithProviders(<AuditLogPage />);

    await waitFor(() => {
      expect(screen.getAllByText('child_delete').length).toBeGreaterThan(0);
    });

    fireEvent.click(screen.getAllByText('child_delete')[0]);

    await waitFor(() => {
      // The dialog renders both an <h2> title and a visually-hidden description
      // with the same i18n key; both are expected.
      expect(screen.getAllByText('auditLog.detailTitle').length).toBeGreaterThan(0);
    });
    // The parsed details JSON should render in the pre block.
    expect(screen.getByText(/"resource_name": "Anna"/)).toBeInTheDocument();
  });

  // An org admin is served the network prefix, never the recorded address. The
  // API says so with ip_anonymized; the dialog used to ignore the flag and
  // print the prefix under a plain "IP address" label, so a truncated value was
  // indistinguishable from an address that happens to end in .0.
  it('labels a redacted IP as a network prefix and shows the correlation fields', async () => {
    (apiClient.getAuditLogs as jest.Mock).mockResolvedValueOnce({
      data: [
        {
          ...mockEntries[0],
          ip_address: '192.168.1.0',
          ip_anonymized: true,
          user_agent: 'Mozilla/5.0 (Kita tablet)',
          request_id: '4b89e4e0-6c37-4e1c-9a78-5d34b2a5f9a1',
        },
      ],
      total: 1,
      page: 1,
      limit: 20,
    });

    renderWithProviders(<AuditLogPage />);

    await waitFor(() => {
      expect(screen.getAllByText('child_delete').length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText('child_delete')[0]);

    await waitFor(() => {
      expect(screen.getByText('auditLog.ipAddressAnonymized')).toBeInTheDocument();
    });
    expect(screen.queryByText('auditLog.ipAddress')).not.toBeInTheDocument();
    expect(screen.getByText('Mozilla/5.0 (Kita tablet)')).toBeInTheDocument();
    expect(screen.getByText('4b89e4e0-6c37-4e1c-9a78-5d34b2a5f9a1')).toBeInTheDocument();
  });

  it('uses the plain IP label when the viewer sees the recorded address', async () => {
    (apiClient.getAuditLogs as jest.Mock).mockResolvedValueOnce({
      data: [{ ...mockEntries[0], ip_address: '203.0.113.7' }],
      total: 1,
      page: 1,
      limit: 20,
    });

    renderWithProviders(<AuditLogPage />);

    await waitFor(() => {
      expect(screen.getAllByText('child_delete').length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText('child_delete')[0]);

    await waitFor(() => {
      expect(screen.getByText('auditLog.ipAddress')).toBeInTheDocument();
    });
    expect(screen.queryByText('auditLog.ipAddressAnonymized')).not.toBeInTheDocument();
  });

  // A refused request is an access-control event. Falling through to the
  // "data change" default would file an attempted intrusion alongside ordinary
  // editing traffic, which is the one place the categories must not be wrong.
  it('files a refused request under access control, not data change', async () => {
    (apiClient.getAuditLogs as jest.Mock).mockResolvedValueOnce({
      data: [
        {
          ...mockEntries[0],
          action: 'access_denied',
          resource_type: '',
          success: false,
          details: '{"reason":"forbidden"}',
        },
      ],
      total: 1,
      page: 1,
      limit: 20,
    });

    renderWithProviders(<AuditLogPage />);

    await waitFor(() => {
      expect(screen.getAllByText('access_denied').length).toBeGreaterThan(0);
    });
    // The category surfaces as the icon's aria-label rather than as text, which
    // is also the only form a screen-reader user gets it in.
    expect(screen.getAllByLabelText('auditLog.categoryAccess').length).toBeGreaterThan(0);
    expect(screen.queryByLabelText('auditLog.categoryData')).not.toBeInTheDocument();
  });
});
