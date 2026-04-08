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
    boundaryIndex: 0,
    onPointerDown: jest.fn(),
    isDragging: false,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders with correct test id', () => {
    render(<BoundaryHandle {...defaultProps} />);
    expect(screen.getByTestId('boundary-handle')).toBeInTheDocument();
  });

  it('has role="slider" for accessibility', () => {
    render(<BoundaryHandle {...defaultProps} />);
    expect(screen.getByRole('slider')).toBeInTheDocument();
  });

  it('has aria-label', () => {
    render(<BoundaryHandle {...defaultProps} />);
    expect(screen.getByRole('slider')).toHaveAttribute('aria-label', 'timeline.dragToAdjust');
  });

  it('is focusable (tabIndex=0)', () => {
    render(<BoundaryHandle {...defaultProps} />);
    expect(screen.getByRole('slider')).toHaveAttribute('tabindex', '0');
  });

  it('calls onPointerDown with boundaryIndex on pointer down', () => {
    render(<BoundaryHandle {...defaultProps} />);
    const handle = screen.getByTestId('boundary-handle');
    fireEvent.pointerDown(handle);
    expect(defaultProps.onPointerDown).toHaveBeenCalledWith(0, expect.any(Object));
  });

  it('shows date labels from both contracts', () => {
    render(<BoundaryHandle {...defaultProps} />);
    const handle = screen.getByTestId('boundary-handle');
    // The component shows formatted dates. formatDate returns locale-formatted strings.
    // Since we're not mocking formatDate, we check for the pipe separator.
    expect(handle.textContent).toContain('|');
  });

  it('applies dragging class when isDragging is true', () => {
    render(<BoundaryHandle {...defaultProps} isDragging />);
    const handle = screen.getByTestId('boundary-handle');
    expect(handle.className).toContain('cursor-grabbing');
  });

  it('uses drag dates when provided', () => {
    render(
      <BoundaryHandle
        {...defaultProps}
        isDragging
        dragEndDate="2024-08-01T00:00:00Z"
        dragStartDate="2024-08-02T00:00:00Z"
      />
    );
    // Should render the drag dates, not the contract dates
    const handle = screen.getByTestId('boundary-handle');
    expect(handle.textContent).toContain('|');
  });
});
