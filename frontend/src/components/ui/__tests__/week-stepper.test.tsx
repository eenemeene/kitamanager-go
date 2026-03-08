import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { WeekStepper } from '../week-stepper';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
  useLocale: () => 'en',
}));

describe('WeekStepper', () => {
  const onChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders navigation buttons', () => {
    renderWithProviders(<WeekStepper value={new Date(2026, 0, 15)} onChange={onChange} />);

    expect(screen.getByLabelText('previousWeek')).toBeInTheDocument();
    expect(screen.getByLabelText('nextWeek')).toBeInTheDocument();
    expect(screen.getByText('thisWeek')).toBeInTheDocument();
  });

  it('calls onChange with previous week when left arrow clicked', async () => {
    const user = userEvent.setup();
    // 2026-01-15 is a Thursday, Monday of that week is Jan 12
    renderWithProviders(<WeekStepper value={new Date(2026, 0, 15)} onChange={onChange} />);

    await user.click(screen.getByLabelText('previousWeek'));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    // Previous week Monday: Jan 5
    expect(calledDate.getDate()).toBe(5);
    expect(calledDate.getMonth()).toBe(0);
  });

  it('calls onChange with next week when right arrow clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<WeekStepper value={new Date(2026, 0, 15)} onChange={onChange} />);

    await user.click(screen.getByLabelText('nextWeek'));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    // Next week Monday: Jan 19
    expect(calledDate.getDate()).toBe(19);
    expect(calledDate.getMonth()).toBe(0);
  });

  it('calls onChange with this week Monday when thisWeek clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<WeekStepper value={new Date(2020, 0, 1)} onChange={onChange} />);

    await user.click(screen.getByText('thisWeek'));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    // Should be a Monday (day 1)
    expect(calledDate.getDay()).toBe(1);
  });
});
