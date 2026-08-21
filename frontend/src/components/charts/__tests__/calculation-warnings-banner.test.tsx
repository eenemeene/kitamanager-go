import { render, screen } from '@testing-library/react';
import { CalculationWarningsBanner } from '../calculation-warnings-banner';
import type { CalculationWarning } from '@/lib/api/types';

function makeWarning(overrides: Partial<CalculationWarning> = {}): CalculationWarning {
  return {
    code: '',
    message: '',
    employee_id: 0,
    contract_id: 0,
    payplan_id: 0,
    date: '',
    grade: '',
    step: 0,
    ...overrides,
  };
}

describe('CalculationWarningsBanner', () => {
  it('renders nothing when warnings is undefined', () => {
    const { container } = render(<CalculationWarningsBanner warnings={undefined} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders nothing when warnings is empty', () => {
    const { container } = render(<CalculationWarningsBanner warnings={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders the banner when warnings is non-empty', () => {
    const w = makeWarning({
      code: 'missing_pay_plan',
      message: 'employee contract references unknown pay plan; salary excluded',
      employee_id: 42,
      contract_id: 99,
      payplan_id: 7,
      date: '2026-03-01',
    });
    render(<CalculationWarningsBanner warnings={[w]} />);
    expect(screen.getByTestId('calculation-warnings-banner')).toBeInTheDocument();
  });

  it('renders one list item per warning (no client-side dedupe)', () => {
    // The backend already de-dupes per (code, contract_id); rendering one
    // item per row is the right shape so the user can see "the same
    // misconfiguration affects employees A and B" rather than collapsing
    // them into a vague "missing_pay_plan" line.
    const warnings: CalculationWarning[] = [
      makeWarning({
        code: 'missing_pay_plan',
        message: 'msg',
        employee_id: 1,
        contract_id: 11,
        payplan_id: 7,
      }),
      makeWarning({
        code: 'missing_pay_plan',
        message: 'msg',
        employee_id: 2,
        contract_id: 22,
        payplan_id: 7,
      }),
      makeWarning({
        code: 'no_pay_plan_period',
        message: 'msg',
        employee_id: 3,
        contract_id: 33,
        payplan_id: 7,
      }),
    ];
    render(<CalculationWarningsBanner warnings={warnings} />);
    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(3);
  });

  it('renders unusable_pay_plan_period via its own translation, not the raw message', () => {
    // The forward-compat fallback below means a new backend code is never
    // broken — it just arrives in English. German is the primary audience
    // here, so a code that ships without its string is a real gap even
    // though nothing crashes.
    const w = makeWarning({
      code: 'unusable_pay_plan_period',
      message: 'pay plan period has invalid weekly hours; salary excluded',
      employee_id: 42,
      contract_id: 99,
      payplan_id: 7,
      date: '2026-03-01',
    });
    render(<CalculationWarningsBanner warnings={[w]} />);
    const item = screen.getByTestId('calculation-warnings-banner');
    expect(item).toHaveTextContent('unusablePayPlanPeriod');
    expect(item).not.toHaveTextContent('pay plan period has invalid weekly hours');
  });

  it('renders unknown codes via the raw backend message (forward compat)', () => {
    // A backend evolution that ships a new code before the frontend
    // adds an i18n entry must NOT crash or render an empty line — fall
    // through to the raw message so the user still gets information.
    const warnings: CalculationWarning[] = [
      makeWarning({
        code: 'newly_invented_code',
        message: 'something wrong with row 7',
        contract_id: 7,
      }),
    ];
    render(<CalculationWarningsBanner warnings={warnings} />);
    expect(screen.getByText(/something wrong with row 7/)).toBeInTheDocument();
  });

  it('groups by code with the most-frequent first', () => {
    // Visual-priority test: ten "missing pay plan" rows should render
    // before one "no_pay_plan_entry" row so the user's eye lands on
    // the most actionable category. Order is by group size, not by
    // input order.
    const warnings: CalculationWarning[] = [
      makeWarning({
        code: 'no_pay_plan_entry',
        message: 'rare',
        employee_id: 99,
        grade: 'X',
        step: 1,
      }),
      ...Array.from({ length: 10 }, (_, i) =>
        makeWarning({
          code: 'missing_pay_plan',
          message: 'common',
          employee_id: i + 1,
          contract_id: i + 100,
          payplan_id: 7,
        })
      ),
    ];
    render(<CalculationWarningsBanner warnings={warnings} />);
    const items = screen.getAllByRole('listitem');
    // First 10 items are the missing_pay_plan group; the rare row is at index 10.
    expect(items[0]!.textContent).toContain('employee #1');
    expect(items[10]!.textContent).toContain('employee #99');
  });

  it('renders contract metadata for missing_pay_plan rows', () => {
    const warnings: CalculationWarning[] = [
      makeWarning({
        code: 'missing_pay_plan',
        message: 'msg',
        employee_id: 42,
        contract_id: 99,
        payplan_id: 7,
        date: '2026-03-01',
      }),
    ];
    render(<CalculationWarningsBanner warnings={warnings} />);
    const item = screen.getByRole('listitem');
    // i18n returns the key in tests; we just verify the metadata bits
    // are spliced in (employee tag and date suffix).
    expect(item.textContent).toContain('employee #42');
    expect(item.textContent).toContain('2026-03-01');
  });
});
