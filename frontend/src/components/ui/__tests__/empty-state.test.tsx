import { render, screen } from '@testing-library/react';
import { Inbox } from 'lucide-react';
import { EmptyState } from '../empty-state';

describe('EmptyState', () => {
  it('renders title only', () => {
    render(<EmptyState title="Nothing here" />);
    expect(screen.getByText('Nothing here')).toBeInTheDocument();
  });

  it('renders title, description, icon and action', () => {
    render(
      <EmptyState
        icon={Inbox}
        title="No items"
        description="Create your first item"
        action={<button>Add</button>}
      />
    );
    expect(screen.getByText('No items')).toBeInTheDocument();
    expect(screen.getByText('Create your first item')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Add' })).toBeInTheDocument();
  });
});
