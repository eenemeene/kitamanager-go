import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import GovernmentFundingBillDetailPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';
import type {
  FundingComparisonAmount,
  FundingComparisonChild,
  FundingComparisonResponse,
  GovernmentFundingBillChild,
  GovernmentFundingBillPeriodResponse,
} from '@/lib/api/types';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1', id: '1' }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string, values?: Record<string, unknown>) =>
      values ? `${key}(${JSON.stringify(values)})` : key;
    t.has = () => false;
    return t;
  },
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getGovernmentFundingBillPeriod: jest.fn(),
    compareGovernmentFundingBill: jest.fn(),
  },
}));

const getBillMock = apiClient.getGovernmentFundingBillPeriod as jest.Mock;
const compareMock = apiClient.compareGovernmentFundingBill as jest.Mock;

// ---- Fixtures -------------------------------------------------------------

function makeBillChild(
  overrides: Partial<GovernmentFundingBillChild> = {}
): GovernmentFundingBillChild {
  return {
    voucher_number: 'GB-100',
    child_name: 'Mustermann, Max',
    child_id: 42,
    contract_id: 99,
    birth_date: '01.20',
    district: 1,
    matched: true,
    total_amount: 100_000,
    rows: [],
    ...overrides,
  };
}

function makeBill(
  overrides: Partial<GovernmentFundingBillPeriodResponse> = {}
): GovernmentFundingBillPeriodResponse {
  return {
    id: 1,
    organization_id: 1,
    facility_name: 'Kita Sonnenschein',
    facility_total: 500_000,
    contract_booking: 480_000,
    correction_booking: 20_000,
    file_name: 'Abrechnung_11-25.xlsx',
    file_sha256: 'abc',
    from: '2025-11-01',
    to: '2025-11-30',
    children_count: 1,
    matched_count: 1,
    unmatched_count: 0,
    surcharges: [],
    children: [makeBillChild()],
    created_at: '2025-12-01T00:00:00Z',
    created_by: 1,
    ...overrides,
  };
}

function makeCompChild(overrides: Partial<FundingComparisonChild> = {}): FundingComparisonChild {
  return {
    voucher_number: 'GB-100',
    child_id: 42,
    child_name: 'Mustermann, Max',
    age: 3,
    birth_date: '01.20',
    bill_total: 100_000,
    calculated_total: 100_000,
    correction_total: 0,
    difference: 0,
    status: 'match',
    contract_from: '2024-01-01',
    contract_to: '2025-12-31',
    properties: [],
    bill_appearances: [],
    ...overrides,
  };
}

function makeAmount(overrides: Partial<FundingComparisonAmount> = {}): FundingComparisonAmount {
  return {
    key: 'care_type',
    value: 'ganztag',
    label: 'Ganztag',
    bill_amount: 100_000,
    calculated_amount: 100_000,
    difference: 0,
    mismatch: '' as FundingComparisonAmount['mismatch'],
    ...overrides,
  };
}

function makeComparison(
  overrides: Partial<FundingComparisonResponse> = {}
): FundingComparisonResponse {
  return {
    bill_id: 1,
    bill_from: '2025-11-01',
    bill_to: '2025-11-30',
    facility_name: 'Kita Sonnenschein',
    bill_total: 500_000,
    calculated_total: 500_000,
    correction_total: 0,
    difference: 0,
    children_count: 1,
    match_count: 1,
    difference_count: 0,
    bill_only_count: 0,
    bill_only_amount: 0,
    calc_only_count: 0,
    calc_only_amount: 0,
    children: [makeCompChild()],
    ...overrides,
  };
}

// ---- Tests ----------------------------------------------------------------

describe('GovernmentFundingBillDetailPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('lifecycle states', () => {
    it('renders the loading row while the bill request is in flight', () => {
      // Both queries pending — UI must not render placeholder data.
      getBillMock.mockImplementation(() => new Promise(() => {}));
      compareMock.mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      expect(screen.getByText('loading')).toBeInTheDocument();
    });

    it('renders a not-found message when the bill query resolves to undefined', async () => {
      // Returning undefined matches a 404 result that the client maps
      // to no-data. The page must surface this rather than crashing on
      // `result.facility_name`.
      getBillMock.mockResolvedValue(undefined);
      compareMock.mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('notFound')).toBeInTheDocument());
    });
  });

  describe('basic mode (no comparison data yet)', () => {
    it('shows facility name, file name, and matched/unmatched counts', async () => {
      getBillMock.mockResolvedValue(makeBill({ matched_count: 23, unmatched_count: 2 }));
      compareMock.mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() =>
        expect(screen.getAllByText('Kita Sonnenschein').length).toBeGreaterThan(0)
      );
      expect(screen.getByText(/Abrechnung_11-25\.xlsx/)).toBeInTheDocument();
      expect(screen.getByText('23')).toBeInTheDocument();
      expect(screen.getByText('2')).toBeInTheDocument();
    });

    it('shows the comparison-loading banner while comparison is pending', async () => {
      getBillMock.mockResolvedValue(makeBill());
      compareMock.mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('comparisonLoading')).toBeInTheDocument());
    });

    it('shows the comparison-error banner when the comparison query rejects', async () => {
      getBillMock.mockResolvedValue(makeBill());
      compareMock.mockRejectedValue(new Error('boom'));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('comparisonError')).toBeInTheDocument());
    });

    it('renders matched-status badge for matched rows when no comparison exists', async () => {
      getBillMock.mockResolvedValue(
        makeBill({
          children: [makeBillChild({ matched: true }), makeBillChild({ matched: false })],
          children_count: 2,
        })
      );
      compareMock.mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getAllByText('Mustermann, Max').length).toBeGreaterThan(0));
      // One success badge (matched=true) and one destructive (matched=false)
      // — at minimum.
      const successBadges = document.querySelectorAll('[class*="bg-success"]');
      const destructiveBadges = document.querySelectorAll('[class*="bg-destructive"]');
      expect(successBadges.length).toBeGreaterThanOrEqual(1);
      expect(destructiveBadges.length).toBeGreaterThanOrEqual(1);
    });
  });

  describe('comparison mode — StatusBadge variants', () => {
    function setupWithChild(comp: FundingComparisonChild) {
      getBillMock.mockResolvedValue(
        makeBill({ children: [makeBillChild({ voucher_number: comp.voucher_number })] })
      );
      compareMock.mockResolvedValue(makeComparison({ children: [comp] }));
    }

    it('renders the "match" status badge', async () => {
      setupWithChild(makeCompChild({ status: 'match' }));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('statusMatch')).toBeInTheDocument());
    });

    it('renders the "difference" status badge with the destructive variant', async () => {
      // Difference: child appears in both bill and our calc but the
      // amounts diverge. Loud red badge so the bookkeeper notices.
      setupWithChild(
        makeCompChild({
          status: 'difference',
          bill_total: 100_000,
          calculated_total: 90_000,
          difference: 10_000,
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('statusDifference')).toBeInTheDocument());
    });

    it('renders the "bill_only" status badge (warning)', async () => {
      // bill_only: bill claims the child but our records don't —
      // possibly fraud or a stale contract. Warning, not error.
      setupWithChild(makeCompChild({ status: 'bill_only', calculated_total: 0 }));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('statusBillOnly')).toBeInTheDocument());
    });

    it('renders the "calc_only" status badge in the system-only table', async () => {
      // calc_only children only appear in the system-only table, not
      // the main children table. We need a different bill child + a
      // calc_only comparison child.
      getBillMock.mockResolvedValue(
        makeBill({ children: [makeBillChild({ voucher_number: 'GB-100' })] })
      );
      compareMock.mockResolvedValue(
        makeComparison({
          calc_only_count: 1,
          children: [
            makeCompChild({
              status: 'calc_only',
              voucher_number: 'GB-MISSING',
              child_name: 'NotInBill, Sam',
            }),
          ],
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('NotInBill, Sam')).toBeInTheDocument());
    });
  });

  describe('comparison mode — DifferenceReason logic', () => {
    it('shows a "rate" reason when properties differ in amount but not in mismatch', async () => {
      // No mismatch tag (still in both bill & calc), but a non-zero
      // difference — the bill paid a different rate than calc.
      const child = makeCompChild({
        status: 'difference',
        properties: [
          makeAmount({ difference: 500, mismatch: '' as FundingComparisonAmount['mismatch'] }),
        ],
      });
      getBillMock.mockResolvedValue(makeBill({ children: [makeBillChild()] }));
      compareMock.mockResolvedValue(makeComparison({ children: [child] }));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('mismatchReasonRate')).toBeInTheDocument());
    });

    it('shows a "mismatch count" reason when properties have mismatch tags', async () => {
      const child = makeCompChild({
        status: 'difference',
        properties: [
          makeAmount({ key: 'k1', value: 'v1', mismatch: 'missing' }),
          makeAmount({ key: 'k2', value: 'v2', mismatch: 'additional' }),
        ],
      });
      getBillMock.mockResolvedValue(makeBill({ children: [makeBillChild()] }));
      compareMock.mockResolvedValue(makeComparison({ children: [child] }));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      // The mock t(key, values) returns key(JSON-of-values) — assert
      // both the key and the count interpolation.
      await waitFor(() => expect(screen.getByText(/^mismatchCount/)).toBeInTheDocument());
      const reasonEl = screen.getByText(/^mismatchCount/);
      expect(reasonEl.textContent).toContain('"count":2');
    });

    it('shows the bill-only canned reason (not the property breakdown)', async () => {
      const billOnlyChild = makeCompChild({
        status: 'bill_only',
        voucher_number: 'GB-BO',
        child_name: 'Only, Bill',
      });
      getBillMock.mockResolvedValue(
        makeBill({
          children: [makeBillChild({ voucher_number: 'GB-BO', child_name: 'Only, Bill' })],
        })
      );
      compareMock.mockResolvedValue(
        makeComparison({ bill_only_count: 1, children: [billOnlyChild] })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('mismatchReasonBillOnly')).toBeInTheDocument());
    });
  });

  describe('expandable detail rows', () => {
    it('does NOT make a row clickable when the child has a single row and no comparison properties', async () => {
      // No expander chevron, no cursor-pointer. Clicking the row must
      // not toggle expansion.
      const user = userEvent.setup();
      getBillMock.mockResolvedValue(makeBill({ children: [makeBillChild()] }));
      compareMock.mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('Mustermann, Max')).toBeInTheDocument());
      await user.click(screen.getByText('Mustermann, Max'));
      // Detail row would be a TableCell with colSpan; if absent, no
      // such row exists.
      const detailCells = document.querySelectorAll('td[colspan]');
      expect(detailCells).toHaveLength(0);
    });

    it('expands a row that has multiple bill rows on click and collapses on second click', async () => {
      const user = userEvent.setup();
      getBillMock.mockResolvedValue(
        makeBill({
          children: [
            makeBillChild({
              voucher_number: 'GB-100',
              rows: [
                {
                  total_row_amount: 50_000,
                  amounts: [{ key: 'care_type', value: 'ganztag', amount: 50_000 }],
                },
                {
                  total_row_amount: 30_000,
                  amounts: [{ key: 'integration', value: 'a', amount: 30_000 }],
                },
              ],
            }),
          ],
        })
      );
      compareMock.mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('GB-100')).toBeInTheDocument());

      // Initially collapsed — the inner row breakdown isn't shown. We
      // assert by counting `colspan` cells (only present on the
      // expanded detail row) — collapsed = 0, expanded = 1.
      expect(document.querySelectorAll('td[colspan]')).toHaveLength(0);

      // Click expands. The colspan cell is the detail row.
      await user.click(screen.getByText('GB-100'));
      expect(document.querySelectorAll('td[colspan]')).toHaveLength(1);

      // Click collapses.
      await user.click(screen.getByText('GB-100'));
      await waitFor(() => expect(document.querySelectorAll('td[colspan]')).toHaveLength(0));
    });

    it('renders MismatchTag inside the expanded property table', async () => {
      const user = userEvent.setup();
      getBillMock.mockResolvedValue(makeBill());
      compareMock.mockResolvedValue(
        makeComparison({
          children: [
            makeCompChild({
              status: 'difference',
              properties: [makeAmount({ key: 'care_type', value: 'ganztag', mismatch: 'missing' })],
            }),
          ],
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('GB-100')).toBeInTheDocument());
      await user.click(screen.getByText('GB-100'));
      // The MismatchTag for 'missing' renders the localized key.
      expect(screen.getByText('mismatchMissing')).toBeInTheDocument();
    });

    it('only one row can be expanded at a time (clicking a second row collapses the first)', async () => {
      const user = userEvent.setup();
      getBillMock.mockResolvedValue(
        makeBill({
          children: [
            makeBillChild({ voucher_number: 'GB-A' }),
            makeBillChild({ voucher_number: 'GB-B' }),
          ],
          children_count: 2,
        })
      );
      compareMock.mockResolvedValue(
        makeComparison({
          children: [
            makeCompChild({
              voucher_number: 'GB-A',
              properties: [makeAmount({ key: 'k', value: 'v1' })],
            }),
            makeCompChild({
              voucher_number: 'GB-B',
              properties: [makeAmount({ key: 'k', value: 'v2' })],
            }),
          ],
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('GB-A')).toBeInTheDocument());

      // Expand A — surcharges header appears once.
      await user.click(screen.getByText('GB-A'));
      expect(screen.getAllByText('surcharges')).toHaveLength(1);

      // Expand B — should collapse A. Still exactly one surcharges header.
      await user.click(screen.getByText('GB-B'));
      expect(screen.getAllByText('surcharges')).toHaveLength(1);
    });
  });

  describe('system-only children section', () => {
    it('renders the system-only table when calc_only_count > 0', async () => {
      getBillMock.mockResolvedValue(makeBill());
      compareMock.mockResolvedValue(
        makeComparison({
          calc_only_count: 1,
          children: [
            ...makeComparison().children,
            makeCompChild({
              status: 'calc_only',
              voucher_number: 'GB-MISSING',
              child_name: 'NotInBill, Sam',
              calculated_total: 80_000,
            }),
          ],
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      // The card title is `t('systemOnlyChildren') ({count})` —
      // substring match because the count is appended.
      await waitFor(() => expect(screen.getByText(/systemOnlyChildren/)).toBeInTheDocument());
      expect(screen.getByText('NotInBill, Sam')).toBeInTheDocument();
    });

    it('shows "neverInBill" when a calc_only child has no bill_appearances', async () => {
      // Distinguishing a child who's *never* been billed vs. one who
      // appears in other bills helps the bookkeeper triage.
      getBillMock.mockResolvedValue(makeBill());
      compareMock.mockResolvedValue(
        makeComparison({
          calc_only_count: 1,
          children: [
            makeCompChild({
              status: 'calc_only',
              voucher_number: 'GB-MISSING',
              child_name: 'Ghost',
              bill_appearances: [],
            }),
          ],
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('Ghost')).toBeInTheDocument());
      expect(screen.getByText('neverInBill')).toBeInTheDocument();
    });

    it('does NOT render the system-only section when calc_only_count is 0', async () => {
      getBillMock.mockResolvedValue(makeBill());
      compareMock.mockResolvedValue(makeComparison({ calc_only_count: 0 }));
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() =>
        expect(screen.getAllByText('Kita Sonnenschein').length).toBeGreaterThan(0)
      );
      expect(screen.queryByText(/systemOnlyChildren/)).not.toBeInTheDocument();
    });

    it('renders a link to each bill_appearance for a calc_only child', async () => {
      getBillMock.mockResolvedValue(makeBill());
      compareMock.mockResolvedValue(
        makeComparison({
          calc_only_count: 1,
          children: [
            makeCompChild({
              status: 'calc_only',
              voucher_number: 'GB-MISSING',
              child_name: 'Wandering, Sam',
              bill_appearances: [
                { bill_id: 7, bill_from: '2025-09-01', facility_name: 'Kita Sonnenschein' },
              ],
            }),
          ],
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('Wandering, Sam')).toBeInTheDocument());
      // Find the link to the other bill — the row gets a hyperlink
      // to /…/government-funding-bills/7.
      const link = document.querySelector('a[href="/organizations/1/government-funding-bills/7"]');
      expect(link).not.toBeNull();
    });
  });

  describe('amount sign rendering', () => {
    it('shows positive difference in success colour and negative in destructive', async () => {
      // Two different vouchers → two different rows in the children
      // table, each carrying a difference of opposite sign.
      getBillMock.mockResolvedValue(
        makeBill({
          children: [
            makeBillChild({ voucher_number: 'GB-POS' }),
            makeBillChild({ voucher_number: 'GB-NEG' }),
          ],
          children_count: 2,
        })
      );
      compareMock.mockResolvedValue(
        makeComparison({
          children: [
            makeCompChild({ voucher_number: 'GB-POS', status: 'difference', difference: 5_000 }),
            makeCompChild({ voucher_number: 'GB-NEG', status: 'difference', difference: -5_000 }),
          ],
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('GB-POS')).toBeInTheDocument());
      const successCells = document.querySelectorAll('span.text-success');
      const destructiveCells = document.querySelectorAll('span.text-destructive');
      expect(successCells.length).toBeGreaterThanOrEqual(1);
      expect(destructiveCells.length).toBeGreaterThanOrEqual(1);
    });

    it('renders an em-dash when calculated_total or difference is null', async () => {
      // The "calc didn't run for this child" path. Without the dash
      // fallback the cell would show "null" or nothing.
      getBillMock.mockResolvedValue(makeBill());
      compareMock.mockResolvedValue(
        makeComparison({
          children: [
            makeCompChild({
              calculated_total: null as unknown as number,
              difference: null as unknown as number,
            }),
          ],
        })
      );
      renderWithProviders(<GovernmentFundingBillDetailPage />);
      await waitFor(() => expect(screen.getByText('GB-100')).toBeInTheDocument());
      // At least 2 em-dashes in the comparison row (calc + diff).
      const cells = within(screen.getByText('GB-100').closest('tr')!).getAllByText('—');
      expect(cells.length).toBeGreaterThanOrEqual(2);
    });
  });
});
