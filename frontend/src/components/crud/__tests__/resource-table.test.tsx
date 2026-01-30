import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { ResourceTable, Column } from '../resource-table';

interface TestItem {
  id: number;
  name: string;
  active: boolean;
}

describe('ResourceTable', () => {
  const testItems: TestItem[] = [
    { id: 1, name: 'Item 1', active: true },
    { id: 2, name: 'Item 2', active: false },
  ];

  const columns: Column<TestItem>[] = [
    { key: 'id', header: 'common.id', render: (item) => item.id },
    { key: 'name', header: 'common.name', render: (item) => item.name, className: 'font-medium' },
    {
      key: 'active',
      header: 'common.status',
      render: (item) => (item.active ? 'Active' : 'Inactive'),
    },
  ];

  const getItemKey = (item: TestItem) => item.id;

  it('renders table headers', () => {
    render(<ResourceTable items={testItems} columns={columns} getItemKey={getItemKey} />);

    expect(screen.getByText('common.id')).toBeInTheDocument();
    expect(screen.getByText('common.name')).toBeInTheDocument();
    expect(screen.getByText('common.status')).toBeInTheDocument();
  });

  it('renders table rows', () => {
    render(<ResourceTable items={testItems} columns={columns} getItemKey={getItemKey} />);

    expect(screen.getByText('Item 1')).toBeInTheDocument();
    expect(screen.getByText('Item 2')).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
    expect(screen.getByText('Inactive')).toBeInTheDocument();
  });

  it('renders loading skeletons when isLoading is true', () => {
    render(
      <ResourceTable
        items={undefined}
        columns={columns}
        getItemKey={getItemKey}
        isLoading={true}
        skeletonRows={3}
      />
    );

    // Should not show table data
    expect(screen.queryByText('Item 1')).not.toBeInTheDocument();

    // Should show skeletons (check for skeleton class)
    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBe(3);
  });

  it('renders empty state when items is empty', () => {
    render(<ResourceTable items={[]} columns={columns} getItemKey={getItemKey} />);

    expect(screen.getByText('common.noResults')).toBeInTheDocument();
  });

  it('renders empty state when items is undefined', () => {
    render(<ResourceTable items={undefined} columns={columns} getItemKey={getItemKey} />);

    expect(screen.getByText('common.noResults')).toBeInTheDocument();
  });

  describe('action buttons', () => {
    it('renders edit button when onEdit is provided', () => {
      const onEdit = jest.fn();

      render(
        <ResourceTable
          items={testItems}
          columns={columns}
          getItemKey={getItemKey}
          onEdit={onEdit}
        />
      );

      const editButtons = screen.getAllByRole('button');
      expect(editButtons.length).toBe(2); // One for each item

      fireEvent.click(editButtons[0]);
      expect(onEdit).toHaveBeenCalledWith(testItems[0]);
    });

    it('renders delete button when onDelete is provided', () => {
      const onDelete = jest.fn();

      render(
        <ResourceTable
          items={testItems}
          columns={columns}
          getItemKey={getItemKey}
          onDelete={onDelete}
        />
      );

      const deleteButtons = screen.getAllByRole('button');
      expect(deleteButtons.length).toBe(2);

      fireEvent.click(deleteButtons[0]);
      expect(onDelete).toHaveBeenCalledWith(testItems[0]);
    });

    it('renders view button when onView is provided', () => {
      const onView = jest.fn();

      render(
        <ResourceTable
          items={testItems}
          columns={columns}
          getItemKey={getItemKey}
          onView={onView}
        />
      );

      const viewButtons = screen.getAllByRole('button');
      expect(viewButtons.length).toBe(2);

      fireEvent.click(viewButtons[0]);
      expect(onView).toHaveBeenCalledWith(testItems[0]);
    });

    it('renders all action buttons when provided', () => {
      const onView = jest.fn();
      const onEdit = jest.fn();
      const onDelete = jest.fn();

      render(
        <ResourceTable
          items={testItems}
          columns={columns}
          getItemKey={getItemKey}
          onView={onView}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      );

      // 3 buttons per row, 2 rows
      const buttons = screen.getAllByRole('button');
      expect(buttons.length).toBe(6);
    });

    it('hides action column when showActions is false', () => {
      render(
        <ResourceTable
          items={testItems}
          columns={columns}
          getItemKey={getItemKey}
          onEdit={jest.fn()}
          showActions={false}
        />
      );

      expect(screen.queryByText('common.actions')).not.toBeInTheDocument();
      expect(screen.queryAllByRole('button').length).toBe(0);
    });

    it('disables action buttons when actionsDisabled is true', () => {
      render(
        <ResourceTable
          items={testItems}
          columns={columns}
          getItemKey={getItemKey}
          onEdit={jest.fn()}
          onDelete={jest.fn()}
          actionsDisabled={true}
        />
      );

      const buttons = screen.getAllByRole('button');
      buttons.forEach((button) => {
        expect(button).toBeDisabled();
      });
    });

    it('uses custom renderActions when provided', () => {
      const renderActions = (item: TestItem) => (
        <button data-testid={`custom-action-${item.id}`}>Custom Action</button>
      );

      render(
        <ResourceTable
          items={testItems}
          columns={columns}
          getItemKey={getItemKey}
          renderActions={renderActions}
        />
      );

      expect(screen.getByTestId('custom-action-1')).toBeInTheDocument();
      expect(screen.getByTestId('custom-action-2')).toBeInTheDocument();
    });
  });

  it('applies column className to cells', () => {
    render(<ResourceTable items={testItems} columns={columns} getItemKey={getItemKey} />);

    const nameCells = screen.getAllByText(/Item \d/);
    nameCells.forEach((cell) => {
      expect(cell.closest('td')).toHaveClass('font-medium');
    });
  });
});
