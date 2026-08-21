import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MonthStepper } from '../month-stepper';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
  useLocale: () => 'en',
}));

describe('MonthStepper', () => {
  const onChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders the current month and year', () => {
    renderWithProviders(<MonthStepper value={new Date(2026, 1, 1)} onChange={onChange} />);

    expect(screen.getByText('1. February 2026')).toBeInTheDocument();
  });

  it('renders navigation buttons', () => {
    renderWithProviders(<MonthStepper value={new Date(2026, 1, 1)} onChange={onChange} />);

    expect(screen.getByRole('button', { name: 'previousYear' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'previousMonth' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'nextMonth' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'nextYear' })).toBeInTheDocument();
    expect(screen.getByText('today')).toBeInTheDocument();
  });

  it('calls onChange with previous month when left arrow clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MonthStepper value={new Date(2026, 1, 1)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'previousMonth' }));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getFullYear()).toBe(2026);
    expect(calledDate.getMonth()).toBe(0); // January
    expect(calledDate.getDate()).toBe(1);
  });

  it('calls onChange with next month when right arrow clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MonthStepper value={new Date(2026, 1, 1)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'nextMonth' }));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getFullYear()).toBe(2026);
    expect(calledDate.getMonth()).toBe(2); // March
    expect(calledDate.getDate()).toBe(1);
  });

  it('calls onChange with Berlin today when Today button clicked', async () => {
    // See day-stepper: this date becomes a query parameter, so it has to be the
    // server's calendar day. Pinned to a moment where Berlin and UTC disagree.
    jest.useFakeTimers({ advanceTimers: true });
    jest.setSystemTime(new Date('2026-07-31T22:30:00Z'));
    try {
      const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime });
      renderWithProviders(<MonthStepper value={new Date(2024, 5, 1)} onChange={onChange} />);

      await user.click(screen.getByText('today'));

      expect(onChange).toHaveBeenCalledTimes(1);
      const calledDate = onChange.mock.calls[0][0] as Date;
      expect(calledDate.getFullYear()).toBe(2026);
      expect(calledDate.getMonth()).toBe(7);
      expect(calledDate.getDate()).toBe(1);
    } finally {
      jest.useRealTimers();
    }
  });

  it('handles year boundary correctly (January to December)', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MonthStepper value={new Date(2026, 0, 1)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'previousMonth' }));

    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getFullYear()).toBe(2025);
    expect(calledDate.getMonth()).toBe(11); // December
  });

  it('handles year boundary correctly (December to January)', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MonthStepper value={new Date(2025, 11, 1)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'nextMonth' }));

    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getFullYear()).toBe(2026);
    expect(calledDate.getMonth()).toBe(0); // January
  });

  // Year-step buttons close the "no way to jump years" feedback —
  // pre-fix the user had to click the month chevron twelve times to
  // get from May 2026 to May 2025.

  it('calls onChange one year earlier when previousYear arrow clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MonthStepper value={new Date(2026, 4, 1)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'previousYear' }));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getFullYear()).toBe(2025);
    expect(calledDate.getMonth()).toBe(4); // May, month preserved
    expect(calledDate.getDate()).toBe(1); // start of month
  });

  it('calls onChange one year later when nextYear arrow clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MonthStepper value={new Date(2026, 4, 1)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'nextYear' }));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getFullYear()).toBe(2027);
    expect(calledDate.getMonth()).toBe(4); // May, month preserved
    expect(calledDate.getDate()).toBe(1);
  });

  // Leap-day safety: addYears in date-fns clamps Feb 29 → Feb 28 on
  // a non-leap target year. Verify so the year-step never silently
  // produces an invalid date.
  it('clamps Feb 29 to Feb 28 when stepping into a non-leap year', async () => {
    const user = userEvent.setup();
    // 2024-02-29 is a leap day; 2025 is not a leap year.
    renderWithProviders(<MonthStepper value={new Date(2024, 1, 29)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'nextYear' }));

    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getFullYear()).toBe(2025);
    expect(calledDate.getMonth()).toBe(1); // still February
    // startOfMonth pulls us back to the 1st regardless of the
    // clamped 28/29; what we care about here is that the result
    // is a valid date in the right month/year.
    expect(calledDate.getDate()).toBe(1);
  });
});
