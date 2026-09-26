import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ChildrenTable } from '../children-table';
import { renderWithProviders } from '@/test-utils';
import type { Child, ChildFundingResponse, ChildBillingSummaryEntry } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => {
    const t = (key: string) => key;
    t.has = () => false;
    return t;
  },
}));

// A child who moved from Krippe to Mäuse on 2026-08-01. Both contracts are on
// record, because the list endpoint preloads the whole history regardless of
// the `active_on` it filtered the rows by.
const moved: Child = {
  id: 1,
  organization_id: 1,
  first_name: 'Emma',
  last_name: 'Schmidt',
  gender: 'female',
  birthdate: '2018-06-15',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  vouchers: [],
  contracts: [
    {
      id: 1,
      child_id: 1,
      version: 1,
      from: '2025-08-01T00:00:00Z',
      to: '2026-07-31T00:00:00Z',
      section_id: 1,
      section_name: 'Krippe',
      properties: {},
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 2,
      child_id: 1,
      version: 1,
      from: '2026-08-01T00:00:00Z',
      to: null,
      section_id: 2,
      section_name: 'Mäuse',
      properties: {},
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
} as unknown as Child;

const noop = () => {};

function renderAt(asOf: string, overrides: Partial<Parameters<typeof ChildrenTable>[0]> = {}) {
  return renderWithProviders(
    <ChildrenTable
      items={[moved]}
      fundingByChildId={new Map<number, ChildFundingResponse>()}
      billingSummaryByChildId={new Map<number, ChildBillingSummaryEntry>()}
      asOf={asOf}
      onViewHistory={noop}
      onViewBilling={noop}
      onAddContract={noop}
      onEdit={noop}
      onDelete={noop}
      onManageVouchers={noop}
      {...overrides}
    />
  );
}

describe('ChildrenTable', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-09-26T09:00:00Z'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  // The roster is filtered server-side by `active_on`, so the row's own values
  // have to be read from the contract covering that date. Reading today's
  // instead put the room a child has since moved to on a row about last spring.
  it('shows the section from the contract in force on the displayed date', () => {
    renderAt('2026-03-01');
    expect(screen.getByText('Krippe')).toBeInTheDocument();
    expect(screen.queryByText('Mäuse')).not.toBeInTheDocument();
  });

  it('shows the current section when the displayed date is today', () => {
    renderAt('2026-09-26');
    expect(screen.getByText('Mäuse')).toBeInTheDocument();
    expect(screen.queryByText('Krippe')).not.toBeInTheDocument();
  });

  // Below sm the four secondary actions are hidden for width, and every page
  // they lead to is reachable from nowhere else in the app -- so the row's menu
  // is the only door to them on a phone, not a convenience.
  it('reaches every hidden action through the row menu', async () => {
    jest.useRealTimers();
    const user = userEvent.setup();
    const onAddContract = jest.fn();
    const onManageVouchers = jest.fn();
    renderAt('2026-09-26', { onAddContract, onManageVouchers });

    const trigger = screen.getByRole('button', { name: 'common.actions' });
    await user.click(trigger);

    expect(await screen.findByRole('menuitem', { name: 'children.contractHistory' })).toBeVisible();
    expect(screen.getByRole('menuitem', { name: 'children.billingHistory' })).toBeVisible();
    expect(screen.getByRole('menuitem', { name: 'vouchers.dialogTitle' })).toBeVisible();

    await user.click(screen.getByRole('menuitem', { name: 'children.addContract' }));
    expect(onAddContract).toHaveBeenCalledWith(moved);
  });

  // The warning and the button beside it say "this contract should already have
  // ended". That is a statement about the record now, not about a row showing
  // some other month — and the button would amend a contract the user is not
  // looking at.
  it('offers the contract-end adjustment only while showing today', () => {
    const onAdjustContractEnd = jest.fn();
    const { unmount } = renderAt('2026-09-26', { orgState: 'berlin', onAdjustContractEnd });
    const onToday = screen.queryAllByLabelText(/children\.adjustContractEnd/).length;
    unmount();

    renderAt('2026-03-01', { orgState: 'berlin', onAdjustContractEnd });
    expect(screen.queryAllByLabelText(/children\.adjustContractEnd/)).toHaveLength(0);

    // Guard the guard: if the fixture stopped producing a warning at all, the
    // assertion above would pass for the wrong reason.
    expect(onToday).toBeGreaterThan(0);
  });
});
