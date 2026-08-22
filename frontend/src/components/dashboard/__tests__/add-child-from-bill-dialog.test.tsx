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

/**
 * Radix's Select is not drivable in jsdom, so the submit path here was never
 * exercised -- and that is exactly where a wire-format bug shipped. Standing it
 * up as a native <select> makes the whole chain testable without a workaround
 * that would obscure the test's intent.
 */
jest.mock('@/components/ui/select', () => {
  const React: typeof import('react') = require('react');
  const Ctx = React.createContext<((v: string) => void) | undefined>(undefined);
  return {
    Select: ({
      children,
      onValueChange,
      value,
    }: {
      children: React.ReactNode;
      onValueChange?: (v: string) => void;
      value?: string;
    }) =>
      React.createElement(
        Ctx.Provider,
        { value: onValueChange },
        React.createElement('div', { 'data-testid': 'select', 'data-value': value }, children)
      ),
    SelectTrigger: ({ children }: { children: React.ReactNode }) =>
      React.createElement('div', null, children),
    SelectValue: () => null,
    SelectContent: ({ children }: { children: React.ReactNode }) =>
      React.createElement('div', null, children),
    SelectItem: ({ children, value }: { children: React.ReactNode; value: string }) => {
      const onValueChange = React.useContext(Ctx);
      return React.createElement(
        'button',
        { type: 'button', onClick: () => onValueChange?.(value) },
        children
      );
    },
  };
});

const billChild = {
  voucher_number: 'GB-12345678901-02',
  child_name: 'Beispiel,Anna',
  first_name: 'Anna',
  last_name: 'Beispiel',
  bill_birth_date: '03.20',
  district: 11,
  first_seen_bill_id: 42,
  first_seen_bill_from: '2025-02-01',
} as UnmatchedBillChild;

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

/**
 * The wire format for the contract's start date.
 *
 * `ChildContractCreateRequest.From` is a Go `time.Time`, which only unmarshals
 * RFC3339 -- and `<input type="date">` yields a bare "YYYY-MM-DD". Sending it
 * raw meant the child was created and the contract was then refused with a 400,
 * leaving a child with no contract and no voucher behind an error toast.
 */
describe('contract start date on the wire', () => {
  async function fillAndSubmit() {
    renderWithProviders(
      <AddChildFromBillDialog open={true} onOpenChange={() => {}} orgId={1} billChild={billChild} />
    );
    // Pick a section, which is the one field the bill cannot pre-fill.
    fireEvent.click(await screen.findByText('Sonnengruppe'));
    fireEvent.click(screen.getByRole('button', { name: 'addChildFromBill.createAction' }));
  }

  it('sends RFC3339, not the bare date the input holds', async () => {
    (apiClient.createChild as jest.Mock).mockResolvedValue({ id: 99 });
    (apiClient.assignChildVoucher as jest.Mock).mockResolvedValue(undefined);

    await fillAndSubmit();

    await waitFor(() => expect(apiClient.createChild).toHaveBeenCalled());
    const [, payload] = (apiClient.createChild as jest.Mock).mock.calls[0];
    expect(payload.contract.from).toBe('2025-02-01T00:00:00Z');
    expect(payload.contract.section_id).toBe(7);
  });

  it('creates the child and contract in one request', async () => {
    // Two requests meant a rejected contract left a childless record behind.
    // The contract rides inside the create so the server can commit both or
    // neither.
    (apiClient.createChild as jest.Mock).mockResolvedValue({ id: 99 });
    (apiClient.assignChildVoucher as jest.Mock).mockResolvedValue(undefined);

    await fillAndSubmit();

    await waitFor(() => expect(apiClient.createChild).toHaveBeenCalled());
    expect(apiClient.createChildContract).not.toHaveBeenCalled();
  });

  it('keeps the child when only the voucher is refused', async () => {
    // Deliberately outside the transaction: a cross-org 409 means the number
    // belongs to someone else, and discarding a correct child and contract over
    // it would be the worse outcome. The user is told, and fixes the voucher
    // from the Vouchers dialog.
    (apiClient.createChild as jest.Mock).mockResolvedValue({ id: 99 });
    (apiClient.assignChildVoucher as jest.Mock).mockRejectedValue(new Error('voucher taken'));

    await fillAndSubmit();

    await waitFor(() =>
      expect(toastMock).toHaveBeenCalledWith(expect.objectContaining({ variant: 'destructive' }))
    );
    expect(apiClient.createChild).toHaveBeenCalledTimes(1);
  });

  it('completes the whole chain and reports success', async () => {
    (apiClient.createChild as jest.Mock).mockResolvedValue({ id: 99 });
    (apiClient.assignChildVoucher as jest.Mock).mockResolvedValue(undefined);

    await fillAndSubmit();

    await waitFor(() => expect(apiClient.assignChildVoucher).toHaveBeenCalled());
    expect(apiClient.assignChildVoucher).toHaveBeenCalledWith(1, 99, billChild.voucher_number);
    await waitFor(() =>
      expect(toastMock).toHaveBeenCalledWith(
        expect.objectContaining({ title: 'addChildFromBill.success' })
      )
    );
  });
});
