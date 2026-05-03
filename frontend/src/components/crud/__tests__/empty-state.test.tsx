import React from 'react';
import { render, screen } from '@testing-library/react';
import { axe } from 'jest-axe';
import { Plus, Baby } from 'lucide-react';
import { EmptyState } from '../empty-state';

describe('EmptyState', () => {
  it('renders title and description as i18n keys when no spaces', () => {
    render(
      <EmptyState icon={Baby} title="children.emptyTitle" description="children.emptyDescription" />
    );
    expect(screen.getByText('children.emptyTitle')).toBeInTheDocument();
    expect(screen.getByText('children.emptyDescription')).toBeInTheDocument();
  });

  it('renders title and description as raw text when they contain spaces', () => {
    render(<EmptyState icon={Baby} title="No data" description="Add something." />);
    expect(screen.getByText('No data')).toBeInTheDocument();
    expect(screen.getByText('Add something.')).toBeInTheDocument();
  });

  it('renders the action node', () => {
    render(
      <EmptyState
        icon={Baby}
        title="No data"
        description="Add something."
        action={<button>Add</button>}
      />
    );
    expect(screen.getByRole('button', { name: 'Add' })).toBeInTheDocument();
  });

  it('renders the icon', () => {
    const { container } = render(<EmptyState icon={Plus} title="No data" description="x." />);
    expect(container.querySelector('svg')).toBeInTheDocument();
  });

  it('has no accessibility violations', async () => {
    const { container } = render(
      <EmptyState icon={Baby} title="No data" description="Add something." />
    );
    expect(await axe(container)).toHaveNoViolations();
  });
});
