import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { axe } from 'jest-axe';
import { CrudPageHeader } from '../crud-page-header';

describe('CrudPageHeader', () => {
  const defaultProps = {
    title: 'items.title',
    onNew: jest.fn(),
    newButtonText: 'items.newItem',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders title', () => {
    render(<CrudPageHeader {...defaultProps} />);

    expect(screen.getByText('items.title')).toBeInTheDocument();
  });

  it('renders "New" button', () => {
    render(<CrudPageHeader {...defaultProps} />);

    expect(screen.getByText('items.newItem')).toBeInTheDocument();
  });

  it('calls onNew when button is clicked', () => {
    render(<CrudPageHeader {...defaultProps} />);

    fireEvent.click(screen.getByText('items.newItem'));

    expect(defaultProps.onNew).toHaveBeenCalled();
  });

  it('hides "New" button when hideNewButton is true', () => {
    render(<CrudPageHeader {...defaultProps} hideNewButton={true} />);

    expect(screen.queryByText('items.newItem')).not.toBeInTheDocument();
  });

  it('disables "New" button when newButtonDisabled is true', () => {
    render(<CrudPageHeader {...defaultProps} newButtonDisabled={true} />);

    expect(screen.getByText('items.newItem').closest('button')).toBeDisabled();
  });

  it('renders title directly when it does not contain a dot', () => {
    render(<CrudPageHeader {...defaultProps} title="My Page Title" />);

    // When title doesn't contain a dot, it's rendered as-is
    expect(screen.getByText('My Page Title')).toBeInTheDocument();
  });

  it('renders button text directly when it does not contain a dot', () => {
    render(<CrudPageHeader {...defaultProps} newButtonText="Add New" />);

    expect(screen.getByText('Add New')).toBeInTheDocument();
  });

  it('renders description as i18n key when no spaces (treated as key)', () => {
    render(<CrudPageHeader {...defaultProps} description="items.description" />);

    expect(screen.getByText('items.description')).toBeInTheDocument();
  });

  it('renders description directly when it contains spaces (raw text)', () => {
    render(<CrudPageHeader {...defaultProps} description="A short subtitle." />);

    expect(screen.getByText('A short subtitle.')).toBeInTheDocument();
  });

  it('does not render description block when description is omitted', () => {
    const { container } = render(<CrudPageHeader {...defaultProps} />);
    // Only the h1 should be present in the title column
    expect(container.querySelectorAll('p').length).toBe(0);
  });

  it('includes Plus icon in button', () => {
    render(<CrudPageHeader {...defaultProps} />);

    // Check the button contains an SVG (the Plus icon)
    const button = screen.getByText('items.newItem').closest('button');
    expect(button?.querySelector('svg')).toBeInTheDocument();
  });

  it('has no accessibility violations', async () => {
    const { container } = render(<CrudPageHeader {...defaultProps} />);
    expect(await axe(container)).toHaveNoViolations();
  });

  it('has no accessibility violations when button is disabled', async () => {
    const { container } = render(<CrudPageHeader {...defaultProps} newButtonDisabled={true} />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
