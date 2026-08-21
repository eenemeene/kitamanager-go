import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { YearStepper } from '../year-stepper';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

describe('YearStepper', () => {
  it('renders current year', () => {
    renderWithProviders(<YearStepper value={2026} onChange={jest.fn()} />);

    expect(screen.getByText('2026')).toBeInTheDocument();
  });

  it('calls onChange with previous year when clicking left button', async () => {
    const onChange = jest.fn();
    renderWithProviders(<YearStepper value={2026} onChange={onChange} />);

    await userEvent.click(screen.getByLabelText('previousYear'));

    expect(onChange).toHaveBeenCalledWith(2025);
  });

  it('calls onChange with next year when clicking right button', async () => {
    const onChange = jest.fn();
    renderWithProviders(<YearStepper value={2026} onChange={onChange} />);

    await userEvent.click(screen.getByLabelText('nextYear'));

    expect(onChange).toHaveBeenCalledWith(2027);
  });

  it('has accessible aria labels on buttons', () => {
    renderWithProviders(<YearStepper value={2026} onChange={jest.fn()} />);

    expect(screen.getByLabelText('previousYear')).toBeInTheDocument();
    expect(screen.getByLabelText('nextYear')).toBeInTheDocument();
  });
});
