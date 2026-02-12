import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { axe } from 'jest-axe';
import { DeleteConfirmDialog } from '../delete-confirm-dialog';

describe('DeleteConfirmDialog', () => {
  const defaultProps = {
    open: true,
    onOpenChange: jest.fn(),
    onConfirm: jest.fn(),
    resourceName: 'items',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders when open', () => {
    render(<DeleteConfirmDialog {...defaultProps} />);

    expect(screen.getByText('common.confirmDelete')).toBeInTheDocument();
    expect(screen.getByText('items.deleteConfirm')).toBeInTheDocument();
    expect(screen.getByText('common.cancel')).toBeInTheDocument();
    expect(screen.getByText('common.delete')).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    render(<DeleteConfirmDialog {...defaultProps} open={false} />);

    expect(screen.queryByText('common.confirmDelete')).not.toBeInTheDocument();
  });

  it('calls onConfirm when delete button is clicked', () => {
    render(<DeleteConfirmDialog {...defaultProps} />);

    fireEvent.click(screen.getByText('common.delete'));

    expect(defaultProps.onConfirm).toHaveBeenCalled();
  });

  it('calls onOpenChange when cancel button is clicked', () => {
    render(<DeleteConfirmDialog {...defaultProps} />);

    fireEvent.click(screen.getByText('common.cancel'));

    expect(defaultProps.onOpenChange).toHaveBeenCalledWith(false);
  });

  it('disables buttons when loading', () => {
    render(<DeleteConfirmDialog {...defaultProps} isLoading={true} />);

    expect(screen.getByText('common.cancel')).toBeDisabled();
    expect(screen.getByText('common.delete')).toBeDisabled();
  });

  it('uses custom description when provided', () => {
    render(
      <DeleteConfirmDialog {...defaultProps} description="Are you sure you want to delete this?" />
    );

    expect(screen.getByText('Are you sure you want to delete this?')).toBeInTheDocument();
    expect(screen.queryByText('items.deleteConfirm')).not.toBeInTheDocument();
  });

  it('applies destructive styling to delete button', () => {
    render(<DeleteConfirmDialog {...defaultProps} />);

    const deleteButton = screen.getByText('common.delete');
    expect(deleteButton).toHaveClass('bg-destructive');
  });

  it('has no accessibility violations when open', async () => {
    const { container } = render(<DeleteConfirmDialog {...defaultProps} />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
