import { screen, waitFor, fireEvent } from '@testing-library/react';
import { UserMembershipDialog } from '../user-membership-dialog';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';
import type { User } from '@/lib/api/types';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getUserMemberships: jest.fn(),
    addUserToOrganization: jest.fn(),
    updateUserOrganizationRole: jest.fn(),
    removeUserFromOrganization: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => {
    if (error && typeof error === 'object' && 'message' in error) {
      return (error as { message: string }).message;
    }
    return fallback;
  }),
}));

const toastMock = jest.fn();
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: toastMock }),
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) {
      const tail = Object.entries(params)
        .map(([k, v]) => `${k}=${v}`)
        .join(',');
      return `${key}(${tail})`;
    }
    return key;
  },
}));

jest.mock('@/stores/ui-store', () => ({
  useUiStore: (selector: (state: unknown) => unknown) =>
    selector({
      organizations: [
        { id: 1, name: 'Kita Sonnenschein' },
        { id: 2, name: 'Kita Mondschein' },
      ],
    }),
}));

const user: User = {
  id: 42,
  name: 'foobar',
  email: 'foobar@example.com',
  active: true,
  is_superadmin: false,
  last_login: '',
  created_at: '2024-01-01T00:00:00Z',
  created_by: 'admin',
  updated_at: '2024-01-01T00:00:00Z',
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe('UserMembershipDialog — add flow', () => {
  it('renders the add form when the user has no membership in the current org', async () => {
    (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({ memberships: [] });

    renderWithProviders(<UserMembershipDialog user={user} orgId={1} onClose={() => {}} />);

    await waitFor(() => {
      expect(screen.getByText(/users\.addToThisOrganization/)).toBeInTheDocument();
    });
    // Org name is interpolated into the prompt.
    expect(
      screen.getByText(/users\.addToThisOrganization.*orgName=Kita Sonnenschein/)
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /users\.addToOrganization/ })).toBeEnabled();
  });

  it('submits with the default role "member" when the button is clicked', async () => {
    (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({ memberships: [] });
    (apiClient.addUserToOrganization as jest.Mock).mockResolvedValue({});

    renderWithProviders(<UserMembershipDialog user={user} orgId={1} onClose={() => {}} />);

    const button = await screen.findByRole('button', { name: /users\.addToOrganization/ });
    fireEvent.click(button);

    await waitFor(() => {
      expect(apiClient.addUserToOrganization).toHaveBeenCalledWith(42, 1, 'member');
    });
  });

  it('surfaces backend error messages via the toast', async () => {
    (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({ memberships: [] });
    (apiClient.addUserToOrganization as jest.Mock).mockRejectedValue({
      message: 'user is already a member of this organization',
    });

    renderWithProviders(<UserMembershipDialog user={user} orgId={1} onClose={() => {}} />);

    const button = await screen.findByRole('button', { name: /users\.addToOrganization/ });
    fireEvent.click(button);

    await waitFor(() => {
      expect(toastMock).toHaveBeenCalledWith(
        expect.objectContaining({
          title: expect.stringMatching(/users\.failedToAddToOrganization/),
          description: 'user is already a member of this organization',
          variant: 'destructive',
        })
      );
    });
  });

  it('does NOT render the add form when the user is already a member of the current org', async () => {
    (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({
      memberships: [{ user_id: 42, organization_id: 1, role: 'admin' }],
    });

    renderWithProviders(<UserMembershipDialog user={user} orgId={1} onClose={() => {}} />);

    // Existing update/remove UI shows roles.role header, not the add prompt.
    await waitFor(() => {
      expect(screen.getByText('roles.role')).toBeInTheDocument();
    });
    expect(screen.queryByText(/users\.addToThisOrganization/)).not.toBeInTheDocument();
  });

  it('lists memberships in other organizations as read-only', async () => {
    (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({
      memberships: [
        {
          user_id: 42,
          organization_id: 2,
          role: 'manager',
          organization: { id: 2, name: 'Kita Mondschein' },
        },
      ],
    });

    renderWithProviders(<UserMembershipDialog user={user} orgId={1} onClose={() => {}} />);

    await waitFor(() => {
      expect(screen.getByText('users.otherMemberships')).toBeInTheDocument();
    });
    expect(screen.getByText('Kita Mondschein')).toBeInTheDocument();
    expect(screen.getByText('roles.manager')).toBeInTheDocument();
  });

  it('interpolates the org name into the add button label', async () => {
    (apiClient.getUserMemberships as jest.Mock).mockResolvedValue({ memberships: [] });

    renderWithProviders(<UserMembershipDialog user={user} orgId={2} onClose={() => {}} />);

    const button = await screen.findByRole('button', { name: /orgName=Kita Mondschein/ });
    expect(button).toBeInTheDocument();
  });
});
