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
});
