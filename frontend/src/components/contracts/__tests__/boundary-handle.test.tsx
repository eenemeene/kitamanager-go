import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { BoundaryHandle } from '../boundary-handle';
import type { BaseContract } from '../timeline-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string) => key;
    t.has = () => false;
    return t;
  },
}));

const upper: BaseContract = { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' };
const lower: BaseContract = { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' };

describe('BoundaryHandle', () => {
  const defaultProps = {
    upperContract: upper,
    lowerContract: lower,
    minDate: new Date('2024-01-01'),
    maxDate: new Date('2024-12-30'),
    onBoundaryChange: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders with correct test id', () => {
    render(<BoundaryHandle {...defaultProps} />);
    expect(screen.getByTestId('boundary-handle')).toBeInTheDocument();
  });

  it('renders as a button', () => {
    render(<BoundaryHandle {...defaultProps} />);
    expect(screen.getByTestId('boundary-handle').tagName).toBe('BUTTON');
  });

  it('has aria-label', () => {
    render(<BoundaryHandle {...defaultProps} />);
    expect(screen.getByTestId('boundary-handle')).toHaveAttribute(
      'aria-label',
      'timeline.clickToAdjust'
    );
  });

  it('shows date labels from both contracts', () => {
    render(<BoundaryHandle {...defaultProps} />);
    const handle = screen.getByTestId('boundary-handle');
    // The component shows formatted dates with a pipe separator
    expect(handle.textContent).toContain('|');
  });

  it('opens calendar popover on click', () => {
    render(<BoundaryHandle {...defaultProps} />);
    fireEvent.click(screen.getByTestId('boundary-handle'));
    // Calendar should be rendered (react-day-picker renders a table)
    expect(screen.getByRole('grid')).toBeInTheDocument();
  });

  it('is disabled when isUpdating is true', () => {
    render(<BoundaryHandle {...defaultProps} isUpdating />);
    expect(screen.getByTestId('boundary-handle')).toBeDisabled();
  });
});
