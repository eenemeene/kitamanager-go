import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FileText, History } from 'lucide-react';
import { RowActionsMenu } from '../row-actions-menu';

describe('RowActionsMenu', () => {
  it('renders nothing when there are no actions', () => {
    const { container } = render(<RowActionsMenu actions={[]} label="Actions for Emma" />);
    expect(container).toBeEmptyDOMElement();
  });

  it('names the row it acts on, so a column of them is distinguishable', () => {
    render(
      <RowActionsMenu
        label="Actions for Emma Schmidt"
        actions={[
          { key: 'history', label: 'Contract history', icon: History, onSelect: jest.fn() },
        ]}
      />
    );
    expect(screen.getByRole('button', { name: 'Actions for Emma Schmidt' })).toBeInTheDocument();
  });

  it('runs the action the user picks', async () => {
    const user = userEvent.setup();
    const addContract = jest.fn();
    render(
      <RowActionsMenu
        label="Actions for Emma Schmidt"
        actions={[
          { key: 'history', label: 'Contract history', icon: History, onSelect: jest.fn() },
          { key: 'add', label: 'Add contract', icon: FileText, onSelect: addContract },
        ]}
      />
    );

    await user.click(screen.getByRole('button', { name: 'Actions for Emma Schmidt' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Add contract' }));

    expect(addContract).toHaveBeenCalledTimes(1);
  });

  it('does not run a disabled action', async () => {
    const user = userEvent.setup();
    const onSelect = jest.fn();
    render(
      <RowActionsMenu
        label="Actions for Emma Schmidt"
        actions={[{ key: 'add', label: 'Add contract', icon: FileText, onSelect, disabled: true }]}
      />
    );

    await user.click(screen.getByRole('button', { name: 'Actions for Emma Schmidt' }));
    const item = await screen.findByRole('menuitem', { name: 'Add contract' });
    await user.click(item);

    expect(onSelect).not.toHaveBeenCalled();
  });
});
