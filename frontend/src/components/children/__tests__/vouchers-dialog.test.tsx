import { screen, waitFor, fireEvent } from '@testing-library/react';
import { VouchersDialog } from '../vouchers-dialog';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';
import type { Child, ChildVoucher } from '@/lib/api/types';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getChildVouchers: jest.fn(),
    assignChildVoucher: jest.fn(),
    removeChildVoucher: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => {
    if (error && typeof error === 'object' && 'message' in error) {
      return (error as { message: string }).message;
    }
    return fallback;
  }),
}));

const toastMock = jest.fn();
jest.mock('@/lib/hooks/use-toast', () => {
  const fn = (args: unknown) => toastMock(args);
  return {
    useToast: () => ({ toast: toastMock }),
    toast: fn,
  };
});

jest.mock('next-intl', () => ({
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

let mockRole: string | null = 'manager';
jest.mock('@/hooks/use-current-role', () => ({
  useCurrentRole: () => mockRole,
  hasMinimumRole: (current: string | null, required: string) => {
    if (!current) return false;
    const order: Record<string, number> = {
      superadmin: 4,
      admin: 3,
      manager: 2,
      member: 1,
      staff: 0,
    };
    return (order[current] ?? -1) >= (order[required] ?? 99);
  },
}));

const child: Child = {
  id: 7,
  organization_id: 1,
  first_name: 'Anna',
  last_name: 'Berger',
  gender: 'female',
  birthdate: '2020-03-15',
  active: true,
  contracts: [],
  vouchers: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
} as Child;

const sampleVouchers: ChildVoucher[] = [
  {
    id: 1,
    child_id: 7,
    voucher_number: 'GB-12345678901-01',
    first_seen: '2025-01-01T00:00:00Z',
  },
  {
    id: 2,
    child_id: 7,
    voucher_number: 'GB-12345678901-02',
    first_seen: '2025-06-01T00:00:00Z',
  },
];

beforeEach(() => {
  jest.clearAllMocks();
  mockRole = 'manager';
});

describe('VouchersDialog — list rendering', () => {
  it('renders all voucher numbers fetched from the API', async () => {
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue(sampleVouchers);

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    await waitFor(() => {
      expect(screen.getByText('GB-12345678901-01')).toBeInTheDocument();
    });
    expect(screen.getByText('GB-12345678901-02')).toBeInTheDocument();
  });

  it('shows an empty-state message when the child has no vouchers', async () => {
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue([]);

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    await waitFor(() => {
      expect(screen.getByText('vouchers.noneAssigned')).toBeInTheDocument();
    });
  });
});

describe('VouchersDialog — add flow', () => {
  it('rejects malformed voucher numbers without hitting the API', async () => {
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue([]);

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    const input = await screen.findByLabelText(/vouchers\.addLabel/);
    fireEvent.change(input, { target: { value: 'totally-wrong' } });
    fireEvent.click(screen.getByRole('button', { name: 'vouchers.addAction' }));

    expect(apiClient.assignChildVoucher).not.toHaveBeenCalled();
    expect(input).toHaveAttribute('aria-invalid', 'true');
  });

  it('calls assignChildVoucher with the trimmed value on a valid number', async () => {
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue([]);
    (apiClient.assignChildVoucher as jest.Mock).mockResolvedValue(undefined);

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    const input = await screen.findByLabelText(/vouchers\.addLabel/);
    fireEvent.change(input, { target: { value: '  GB-12345678901-03  ' } });
    fireEvent.click(screen.getByRole('button', { name: 'vouchers.addAction' }));

    await waitFor(() => {
      expect(apiClient.assignChildVoucher).toHaveBeenCalledWith(1, 7, 'GB-12345678901-03');
    });
  });

  it('shows a destructive toast carrying the backend error message on conflict (409)', async () => {
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue([]);
    (apiClient.assignChildVoucher as jest.Mock).mockRejectedValue({
      message: 'voucher GB-12345678901-03 is already assigned to another child',
    });

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    const input = await screen.findByLabelText(/vouchers\.addLabel/);
    fireEvent.change(input, { target: { value: 'GB-12345678901-03' } });
    fireEvent.click(screen.getByRole('button', { name: 'vouchers.addAction' }));

    await waitFor(() => {
      expect(toastMock).toHaveBeenCalledWith(
        expect.objectContaining({
          variant: 'destructive',
          description: 'voucher GB-12345678901-03 is already assigned to another child',
        })
      );
    });
  });
});

describe('VouchersDialog — remove flow', () => {
  it('opens the confirmation dialog and only calls removeChildVoucher after confirm', async () => {
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue(sampleVouchers);
    (apiClient.removeChildVoucher as jest.Mock).mockResolvedValue(undefined);

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    const removeButton = await screen.findByRole('button', {
      name: 'vouchers.removeAction(number=GB-12345678901-01)',
    });
    fireEvent.click(removeButton);

    // Confirmation dialog appears; nothing called yet.
    await screen.findByText('vouchers.removeConfirmTitle');
    expect(apiClient.removeChildVoucher).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'common.delete' }));

    await waitFor(() => {
      expect(apiClient.removeChildVoucher).toHaveBeenCalledWith(1, 7, 1);
    });
  });
});

describe('VouchersDialog — role-gated UI', () => {
  it('hides Add input and Remove buttons for member role', async () => {
    mockRole = 'member';
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue(sampleVouchers);

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    // List is visible.
    await waitFor(() => {
      expect(screen.getByText('GB-12345678901-01')).toBeInTheDocument();
    });
    // Controls are not.
    expect(screen.queryByLabelText(/vouchers\.addLabel/)).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'vouchers.removeAction(number=GB-12345678901-01)' })
    ).not.toBeInTheDocument();
  });

  it('hides Add and Remove for staff role', async () => {
    mockRole = 'staff';
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue(sampleVouchers);

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    await waitFor(() => {
      expect(screen.getByText('GB-12345678901-01')).toBeInTheDocument();
    });
    expect(screen.queryByLabelText(/vouchers\.addLabel/)).not.toBeInTheDocument();
  });

  it('shows Add and Remove for admin role', async () => {
    mockRole = 'admin';
    (apiClient.getChildVouchers as jest.Mock).mockResolvedValue(sampleVouchers);

    renderWithProviders(
      <VouchersDialog open={true} onOpenChange={() => {}} orgId={1} child={child} />
    );

    expect(await screen.findByLabelText(/vouchers\.addLabel/)).toBeInTheDocument();
  });
});
