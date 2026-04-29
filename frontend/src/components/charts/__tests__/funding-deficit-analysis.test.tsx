import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Table, TableBody } from '@/components/ui/table';
import { FundingDeficitAnalysis } from '../funding-deficit-analysis';
import type {
  FundingComparisonCategorySummary,
  FundingComparisonIssueSummary,
  FundingComparisonSummary,
} from '@/lib/api/types';

// FundingDeficitAnalysis is a TableRow component, so it must render
// inside <Table><TableBody>...</TableBody></Table> — without that,
// React warns and queries fail in odd ways. Wrap each test in this
// utility.
function renderInTableBody(ui: React.ReactNode) {
  return render(
    <Table>
      <TableBody>{ui}</TableBody>
    </Table>
  );
}

function makeCategory(
  overrides: Partial<FundingComparisonCategorySummary> = {}
): FundingComparisonCategorySummary {
  return {
    category: 'rate_difference',
    total_amount: 0,
    child_count: 0,
    actionable: false,
    ...overrides,
  };
}

function makeIssue(
  overrides: Partial<FundingComparisonIssueSummary> = {}
): FundingComparisonIssueSummary {
  return {
    category: 'property_mismatch',
    issue_type: 'missing',
    property_key: 'integration',
    bill_value: '',
    calc_value: 'integration b',
    child_id: 42,
    child_name: 'Bagus, Nathan',
    voucher_number: 'GB-1',
    description: 'integration:b — in contract but not billed',
    amount_per_month: -33064,
    month_count: 12,
    total_amount: -396768,
    actionable: true,
    ...overrides,
  };
}

function makeSummary(overrides: Partial<FundingComparisonSummary> = {}): FundingComparisonSummary {
  return {
    month_count: 12,
    total_billed: 0,
    total_calculated: 0,
    total_corrections: 0,
    total_difference: 0,
    categories: [],
    issues: [],
    ...overrides,
  };
}

describe('FundingDeficitAnalysis', () => {
  describe('empty / no-data branches', () => {
    it('renders nothing (returns null) when there are no categories', () => {
      // Section is hidden entirely if there are no category breakdowns.
      // The early return on line 50-52.
      const { container } = renderInTableBody(
        <FundingDeficitAnalysis summary={makeSummary({ categories: [] })} orgId="1" />
      );
      // The wrapping table renders the colgroup and tbody but no row
      // for the deficit analysis itself.
      expect(container.querySelector('button')).toBeNull();
      expect(screen.queryByText('deficitAnalysis')).not.toBeInTheDocument();
    });

    it('handles undefined categories field (older payloads) by defaulting to []', () => {
      // The component does `summary.categories ?? []` — if the API
      // version skipped this field, it should still not crash.
      const { container } = renderInTableBody(
        <FundingDeficitAnalysis
          summary={
            { ...makeSummary(), categories: undefined } as unknown as FundingComparisonSummary
          }
          orgId="1"
        />
      );
      expect(container.querySelector('button')).toBeNull();
    });
  });

  describe('expand/collapse', () => {
    it('starts collapsed by default and expands on click', async () => {
      const user = userEvent.setup();
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: -100 })],
          })}
          orgId="1"
        />
      );
      // Header button is visible, content is not
      const trigger = screen.getByRole('button', { name: /deficitAnalysis/ });
      expect(trigger).toBeInTheDocument();
      expect(screen.queryByText('deficitCategories')).not.toBeInTheDocument();

      await user.click(trigger);
      expect(screen.getByText('deficitCategories')).toBeInTheDocument();

      await user.click(trigger);
      expect(screen.queryByText('deficitCategories')).not.toBeInTheDocument();
    });

    it('starts expanded when forceExpanded is true (PDF/print path)', () => {
      // Print/PDF rendering needs everything visible without
      // simulating clicks — forceExpanded short-circuits both the
      // section toggle AND the issue-list "show all" toggle.
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: -100 })],
          })}
          orgId="1"
          forceExpanded
        />
      );
      expect(screen.getByText('deficitCategories')).toBeInTheDocument();
    });
  });

  describe('category breakdown', () => {
    it('renders one row per category with localized label', () => {
      // Each category gets translated via t(`deficitCategory_${category}`) —
      // we verify the translation key is hit (jest.setup mock returns key
      // as-is, so the key text appears).
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [
              makeCategory({ category: 'rate_difference', total_amount: -100 }),
              makeCategory({ category: 'property_mismatch', total_amount: 200 }),
            ],
          })}
          orgId="1"
          forceExpanded
        />
      );
      expect(screen.getByText('deficitCategory_rate_difference')).toBeInTheDocument();
      expect(screen.getByText('deficitCategory_property_mismatch')).toBeInTheDocument();
    });

    it('shows positive amounts with a leading + and success colour', () => {
      // The "+" prefix is appended only when amount >= 0 — distinguishes
      // overpayment surpluses from invisible-positive numbers.
      const { container } = renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: 12345 })],
          })}
          orgId="1"
          forceExpanded
        />
      );
      const span = Array.from(container.querySelectorAll('span')).find((el) =>
        /^\+\d/.test(el.textContent ?? '')
      );
      expect(span).toBeDefined();
      expect(span!).toHaveClass('text-success');
    });

    it('shows negative amounts without a + and with destructive colour', () => {
      const { container } = renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: -12345 })],
          })}
          orgId="1"
          forceExpanded
        />
      );
      const span = Array.from(container.querySelectorAll('span')).find((el) =>
        el.textContent?.startsWith('-')
      );
      expect(span).toBeDefined();
      expect(span!).toHaveClass('text-destructive');
    });

    it('handles total_amount === 0 as non-negative (text-success, leading +)', () => {
      // Boundary case: 0 must hit the >= 0 branch (success) so categories
      // that net to zero render visibly green-tinted as "+0,00 €". A
      // change to `> 0` would silently flip the colour for these.
      const { container } = renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'x', total_amount: 0 })],
          })}
          orgId="1"
          forceExpanded
        />
      );
      const successSpan = container.querySelector('.text-success');
      expect(successSpan).not.toBeNull();
      expect(successSpan?.textContent).toMatch(/^\+/);
    });

    it('handles undefined total_amount as 0 (no crash on older payloads)', () => {
      const { container } = renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [
              {
                category: 'x',
                child_count: 0,
                actionable: false,
                // total_amount intentionally missing
              } as unknown as FundingComparisonCategorySummary,
            ],
          })}
          orgId="1"
          forceExpanded
        />
      );
      // No throw; success colour applied (0 is non-negative).
      expect(container.querySelector('.text-success')).not.toBeNull();
    });

    it('scales bar width relative to the largest absolute total_amount', () => {
      // Bar width is (|amount| / maxAbs) * 100%. With maxAbs=200 and
      // amount=-100, the bar should be 50% wide.
      const { container } = renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [
              makeCategory({ category: 'a', total_amount: -100 }),
              makeCategory({ category: 'b', total_amount: 200 }),
            ],
          })}
          orgId="1"
          forceExpanded
        />
      );
      const bars = container.querySelectorAll<HTMLElement>('.h-2.rounded-full > div');
      // Two bars rendered; second should be 100% wide, first should be 50%.
      expect(bars).toHaveLength(2);
      expect(bars[0]?.style.width).toBe('50%');
      expect(bars[1]?.style.width).toBe('100%');
    });

    it('avoids divide-by-zero when ALL categories have total_amount=0 (maxAbs floors at 1)', () => {
      // The `Math.max(..., 1)` floor exists exactly to keep `(0/0)*100`
      // from producing NaN bars. Test pins that defence.
      const { container } = renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'x', total_amount: 0 })],
          })}
          orgId="1"
          forceExpanded
        />
      );
      const bar = container.querySelector<HTMLElement>('.h-2.rounded-full > div');
      // 0 / 1 * 100 = 0 → width 0%
      expect(bar?.style.width).toBe('0%');
    });
  });

  describe('actionable issues table', () => {
    it('renders one row per actionable issue and excludes non-actionable', () => {
      // Only `actionable: true` issues appear in the deficit table —
      // non-actionable ones (e.g. info-only diffs) are hidden because
      // the user can't do anything about them.
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: -1 })],
            issues: [
              makeIssue({ child_name: 'Visible Child', actionable: true }),
              makeIssue({ child_name: 'Hidden Child', actionable: false }),
            ],
          })}
          orgId="1"
          forceExpanded
        />
      );
      expect(screen.getByText('Visible Child')).toBeInTheDocument();
      expect(screen.queryByText('Hidden Child')).not.toBeInTheDocument();
    });

    it('truncates the issue list to 10 by default and shows a "show all" toggle', async () => {
      const user = userEvent.setup();
      const issues = Array.from({ length: 15 }, (_, i) =>
        makeIssue({ child_name: `Child ${i}`, actionable: true })
      );
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: -1 })],
            issues,
          })}
          orgId="1"
          forceExpanded={false}
        />
      );
      // expand the section first
      await user.click(screen.getByRole('button', { name: /deficitAnalysis/ }));

      // Default: 10 visible
      expect(screen.getByText('Child 0')).toBeInTheDocument();
      expect(screen.getByText('Child 9')).toBeInTheDocument();
      expect(screen.queryByText('Child 10')).not.toBeInTheDocument();

      // "Show all (15)" toggle reveals the rest
      const toggle = screen.getByRole('button', { name: /deficitShowAll/ });
      await user.click(toggle);
      expect(screen.getByText('Child 14')).toBeInTheDocument();

      // Collapse back
      await user.click(screen.getByRole('button', { name: /deficitShowLess/ }));
      expect(screen.queryByText('Child 14')).not.toBeInTheDocument();
    });

    it('does NOT show the "show all" toggle when issue count is at the threshold', async () => {
      const user = userEvent.setup();
      // Exactly 10 issues — toggle should be hidden because there are no
      // additional issues to show.
      const issues = Array.from({ length: 10 }, (_, i) =>
        makeIssue({ child_name: `c${i}`, actionable: true })
      );
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: -1 })],
            issues,
          })}
          orgId="1"
        />
      );
      await user.click(screen.getByRole('button', { name: /deficitAnalysis/ }));
      expect(screen.queryByRole('button', { name: /deficitShowAll/ })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /deficitShowLess/ })).not.toBeInTheDocument();
    });

    it('renders a no-issues fallback when there are no actionable issues', async () => {
      const user = userEvent.setup();
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: -1 })],
            issues: [makeIssue({ actionable: false })],
          })}
          orgId="1"
        />
      );
      await user.click(screen.getByRole('button', { name: /deficitAnalysis/ }));
      expect(screen.getByText('deficitNoIssues')).toBeInTheDocument();
    });

    it('renders child name as a link when child_id is set', () => {
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'x', total_amount: -1 })],
            issues: [makeIssue({ child_id: 99, child_name: 'Linked', actionable: true })],
          })}
          orgId="42"
          forceExpanded
        />
      );
      const link = screen.getByRole('link', { name: 'Linked' });
      expect(link).toHaveAttribute('href', '/organizations/42/children/99');
    });

    it('renders child name as plain text (no link) when child_id is missing', () => {
      // child_id may be 0 / undefined for issues that aren't tied to a
      // specific child (e.g. an aggregate billing-period mismatch).
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'x', total_amount: -1 })],
            issues: [makeIssue({ child_id: 0, child_name: 'NoChild', actionable: true })],
          })}
          orgId="1"
          forceExpanded
        />
      );
      // No link element with that name; the name is in a <span>.
      expect(screen.queryByRole('link', { name: 'NoChild' })).not.toBeInTheDocument();
      expect(screen.getByText('NoChild')).toBeInTheDocument();
    });

    it('shows the actionable count as a Badge next to the section toggle', () => {
      renderInTableBody(
        <FundingDeficitAnalysis
          summary={makeSummary({
            categories: [makeCategory({ category: 'rate_difference', total_amount: -1 })],
            issues: [
              makeIssue({ actionable: true }),
              makeIssue({ actionable: true }),
              makeIssue({ actionable: false }), // not counted
            ],
          })}
          orgId="1"
        />
      );
      // jest.setup mocks t to return key, so the badge text is the
      // translation key. With `count: 2` interpolation lost in the
      // mock, just assert the key is present.
      expect(screen.getByText('deficitActionableCount')).toBeInTheDocument();
    });
  });
});
