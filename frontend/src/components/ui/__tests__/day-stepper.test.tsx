import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DayStepper } from '../day-stepper';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
  useLocale: () => 'en',
}));

describe('DayStepper', () => {
  const onChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders navigation buttons', () => {
    renderWithProviders(<DayStepper value={new Date(2026, 0, 15)} onChange={onChange} />);

    expect(screen.getByRole('button', { name: 'previousDay' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'nextDay' })).toBeInTheDocument();
    expect(screen.getByText('today')).toBeInTheDocument();
  });

  it('calls onChange with previous day when left arrow clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DayStepper value={new Date(2026, 0, 15)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'previousDay' }));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getDate()).toBe(14);
    expect(calledDate.getMonth()).toBe(0);
  });

  it('calls onChange with next day when right arrow clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DayStepper value={new Date(2026, 0, 15)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'nextDay' }));

    expect(onChange).toHaveBeenCalledTimes(1);
    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getDate()).toBe(16);
    expect(calledDate.getMonth()).toBe(0);
  });

  it('calls onChange with Berlin today when today button clicked', async () => {
    // Berlin's today, not the browser's: the date this produces becomes an
    // `active_on` query parameter, and the server decides what is active with
    // `models.Today()`. Pinned to a moment where the two disagree -- 00:30 on 1
    // August in Berlin is still 31 July in UTC -- so a browser-clock regression
    // fails here rather than only for users in the wrong timezone.
    jest.useFakeTimers({ advanceTimers: true });
    jest.setSystemTime(new Date('2026-07-31T22:30:00Z'));
    try {
      const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime });
      renderWithProviders(<DayStepper value={new Date(2020, 0, 1)} onChange={onChange} />);

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

  it('crosses month boundary going backward', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DayStepper value={new Date(2026, 1, 1)} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'previousDay' }));

    const calledDate = onChange.mock.calls[0][0] as Date;
    expect(calledDate.getMonth()).toBe(0); // January
    expect(calledDate.getDate()).toBe(31);
  });
});
