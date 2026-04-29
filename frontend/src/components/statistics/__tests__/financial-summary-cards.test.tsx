import { render, screen } from '@testing-library/react';
import { TooltipProvider } from '@/components/ui/tooltip';
import { FinancialSummaryCards } from '../financial-summary-cards';

// FinancialSummaryCards displays three KPI tiles. Tests cover:
//   - basic render with positive values
//   - currency formatting (cents → de-DE Euro)
//   - balance colour switching at the >=0 boundary (negative gets
//     destructive styling, exactly-zero gets info styling)
//   - zero / very large / undefined-via-page-default inputs
//   - the data-visual-mask hooks the e2e visual-regression suite
//     depends on. This test guards them so a refactor that drops the
//     attribute is caught at unit-test time, not by flaky pixel diffs.

function renderCards(props: { totalIncome: number; totalExpenses: number; balance: number }) {
  return render(
    <TooltipProvider>
      <FinancialSummaryCards {...props} />
    </TooltipProvider>
  );
}

describe('FinancialSummaryCards', () => {
  describe('happy path', () => {
    it('renders three tiles with the expected labels', () => {
      renderCards({ totalIncome: 100000, totalExpenses: 50000, balance: 50000 });
      // The next-intl mock returns bare keys.
      expect(screen.getByText('statistics.totalIncome')).toBeInTheDocument();
      expect(screen.getByText('statistics.totalExpenses')).toBeInTheDocument();
      expect(screen.getByText('statistics.balance')).toBeInTheDocument();
    });

    it('formats cents as German EUR strings', () => {
      renderCards({ totalIncome: 166847, totalExpenses: 50000, balance: 116847 });
      // formatCurrency divides cents by 100; de-DE locale uses '.' as
      // thousands separator, ',' as decimal separator, and a NBSP
      // before the € sign. Match the loose form to stay locale-jitter
      // resilient.
      expect(screen.getByText(/1\.668,47/)).toBeInTheDocument();
      expect(screen.getByText(/1\.168,47/)).toBeInTheDocument();
      expect(screen.getByText(/500,00/)).toBeInTheDocument();
    });
  });

  describe('balance styling boundary', () => {
    it('shows positive balance with text-info', () => {
      const { container } = renderCards({ totalIncome: 100, totalExpenses: 0, balance: 100 });
      const balanceTile = container.querySelectorAll('[data-visual-mask="currency"]')[2]!;
      expect(balanceTile).toHaveClass('text-info');
      expect(balanceTile).not.toHaveClass('text-destructive');
    });

    it('treats balance of exactly 0 as non-negative (text-info)', () => {
      // Edge case: 0 is mathematically the boundary; the component uses
      // `balance >= 0 ? 'text-info' : 'text-destructive'`. A change to
      // `>` would silently flip the styling for zero balances, which
      // are common at month boundaries.
      const { container } = renderCards({ totalIncome: 100, totalExpenses: 100, balance: 0 });
      const balanceTile = container.querySelectorAll('[data-visual-mask="currency"]')[2]!;
      expect(balanceTile).toHaveClass('text-info');
    });

    it('shows negative balance with text-destructive', () => {
      const { container } = renderCards({ totalIncome: 50, totalExpenses: 200, balance: -150 });
      const balanceTile = container.querySelectorAll('[data-visual-mask="currency"]')[2]!;
      expect(balanceTile).toHaveClass('text-destructive');
      expect(balanceTile).not.toHaveClass('text-info');
    });
  });

  describe('numeric edge cases', () => {
    it('renders zeroes as 0,00 €, not as a dash', () => {
      // formatCurrency renders `null`/`undefined` as '-'. Zero must NOT
      // hit that branch — it should render as a real currency string,
      // since "no expenses yet" is a meaningful display.
      renderCards({ totalIncome: 0, totalExpenses: 0, balance: 0 });
      const zeroes = screen.getAllByText(/0,00/);
      expect(zeroes.length).toBe(3);
      expect(screen.queryByText('-')).not.toBeInTheDocument();
    });

    it('handles very large amounts without truncating', () => {
      // 999.999.999,99 € — about a billion euros, well above any
      // realistic Kita budget but inside JS safe-integer range. If
      // Intl.NumberFormat ever changes default fractional digits this
      // test catches the regression.
      renderCards({
        totalIncome: 99_999_999_999,
        totalExpenses: 0,
        balance: 99_999_999_999,
      });
      // Two tiles match (income + balance). de-DE formats this as
      // "999.999.999,99" with grouping dots; assert with regex that
      // tolerates locale variations (NBSP vs space before €).
      const matches = screen.getAllByText(/999\.999\.999,99/);
      expect(matches).toHaveLength(2);
    });

    it('handles negative income (refund scenario)', () => {
      // Unusual but possible: a corrective refund could push monthly
      // total_income negative. The tile should still render (and not
      // crash on Intl.NumberFormat receiving a negative).
      renderCards({ totalIncome: -1000, totalExpenses: 5000, balance: -6000 });
      expect(screen.getByText(/-10,00/)).toBeInTheDocument();
      expect(screen.getByText(/^50,00/)).toBeInTheDocument();
      expect(screen.getByText(/-60,00/)).toBeInTheDocument();
    });
  });

  describe('visual regression masking contract', () => {
    // These attributes are referenced by e2e/visual-regression.spec.ts
    // via `[data-visual-mask]`. Dropping them would reintroduce the
    // pixel-diff flakiness documented in that file's header.
    it('marks each currency value with data-visual-mask="currency"', () => {
      const { container } = renderCards({ totalIncome: 1, totalExpenses: 1, balance: 0 });
      const masked = container.querySelectorAll('[data-visual-mask="currency"]');
      expect(masked).toHaveLength(3);
    });
  });
});
