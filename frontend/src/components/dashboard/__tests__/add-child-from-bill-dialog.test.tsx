import { screen, waitFor, fireEvent } from '@testing-library/react';
import { AddChildFromBillDialog } from '../add-child-from-bill-dialog';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';
import type { UnmatchedBillChild } from '@/lib/api/types';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSections: jest.fn(),
    createChild: jest.fn(),
    createChildContract: jest.fn(),
    assignChildVoucher: jest.fn(),
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

const billChild: UnmatchedBillChild = {
  voucher_number: 'GB-12345678901-02',
  child_name: 'Beispiel,Anna',
  first_name: 'Anna',
  last_name: 'Beispiel',
  bill_birth_date: '03.20',
  district: 11,
  first_seen_bill_id: 42,
  first_seen_bill_from: '2025-02-01',
};

const sections = createMockPaginatedResponse([
  { id: 7, name: 'Sonnengruppe', organization_id: 1, created_at: '', updated_at: '' },
  { id: 8, name: 'Mondgruppe', organization_id: 1, created_at: '', updated_at: '' },
]);

beforeEach(() => {
  jest.clearAllMocks();
  (apiClient.getSections as jest.Mock).mockResolvedValue(sections);
});

describe('AddChildFromBillDialog — prefill', () => {
  it('pre-fills first/last name + birthdate + contract start from bill', async () => {
    renderWithProviders(
      <AddChildFromBillDialog open={true} onOpenChange={() => {}} orgId={1} billChild={billChild} />
    );

    const firstName = await screen.findByLabelText(/children\.firstName/);
    const lastName = await screen.findByLabelText(/children\.lastName/);
    const birthdate = await screen.findByLabelText(/children\.birthdate/);
    const contractFrom = await screen.findByLabelText(/contracts\.startDate/);

    expect(firstName).toHaveValue('Anna');
    expect(lastName).toHaveValue('Beispiel');
    // First day of the bill's first-seen month.
    expect(birthdate).toHaveValue('2025-02-01');
    expect(contractFrom).toHaveValue('2025-02-01');
  });

  it('shows the birthdate-placeholder warning banner', async () => {
    renderWithProviders(
      <AddChildFromBillDialog open={true} onOpenChange={() => {}} orgId={1} billChild={billChild} />
    );

    expect(await screen.findByText('addChildFromBill.birthdateWarning')).toBeInTheDocument();
  });
});

describe('AddChildFromBillDialog — submit gate', () => {
  // The Create button is gated on section_id > 0. Without picking a
  // section the button stays disabled, so we know the dialog won't
  // accidentally submit half-prefilled data. Radix's Select dropdown
  // relies on pointer events that don't reproduce reliably in jsdom,
  // so we test the gate at the button level rather than driving the
  // full Radix interaction. The chained submit logic is covered by
  // the implementation directly and the existing service-layer tests
  // for the underlying endpoints.
  it('Create button is disabled until a section is picked', async () => {
    renderWithProviders(
      <AddChildFromBillDialog open={true} onOpenChange={() => {}} orgId={1} billChild={billChild} />
    );

    const createBtn = await screen.findByRole('button', {
      name: 'addChildFromBill.createAction',
    });
    expect(createBtn).toBeDisabled();
    // Voucher → children → bills mutations must NOT have fired yet.
    expect(apiClient.createChild).not.toHaveBeenCalled();
    expect(apiClient.createChildContract).not.toHaveBeenCalled();
    expect(apiClient.assignChildVoucher).not.toHaveBeenCalled();
  });

  it('Create button is also disabled when first_name is cleared', async () => {
    renderWithProviders(
      <AddChildFromBillDialog open={true} onOpenChange={() => {}} orgId={1} billChild={billChild} />
    );

    const firstName = await screen.findByLabelText(/children\.firstName/);
    fireEvent.change(firstName, { target: { value: '' } });

    const createBtn = screen.getByRole('button', { name: 'addChildFromBill.createAction' });
    expect(createBtn).toBeDisabled();
  });
});

// Note: the success path (createChild → createChildContract → assignChildVoucher
// chain) and the failure path (voucher 409) are intentionally covered by the
// dialog's wiring + the underlying service tests rather than by exercising
// the Radix Select dropdown here. waitFor + fireEvent doesn't reliably drive
// Radix Select in jsdom, and adding a manual workaround for one widget
// component would obscure the test intent.
