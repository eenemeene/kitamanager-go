import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AttendanceTable, type AttendanceRow } from '../attendance-table';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

const mockOnQuickStatus = jest.fn();
const mockOnEdit = jest.fn();
const mockOnDelete = jest.fn();

const baseRow: AttendanceRow = {
  childId: 1,
  childName: 'Alice Smith',
  attendance: null,
};

const rowWithAttendance: AttendanceRow = {
  childId: 2,
  childName: 'Bob Jones',
  attendance: {
    id: 10,
    child_id: 2,
    organization_id: 1,
    date: '2024-01-15',
    status: 'present',
    check_in_time: '2024-01-15T08:30:00Z',
    check_out_time: '2024-01-15T16:00:00Z',
    note: 'Arrived early',
    recorded_by: 1,
    created_at: '2024-01-15T08:00:00Z',
    updated_at: '2024-01-15T08:00:00Z',
  },
};

describe('AttendanceTable', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders empty state when no rows', () => {
    renderWithProviders(
      <AttendanceTable
        rows={[]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    expect(screen.getByText('noChildren')).toBeInTheDocument();
  });

  it('renders child name and "notRecorded" when no attendance', () => {
    renderWithProviders(
      <AttendanceTable
        rows={[baseRow]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    expect(screen.getByText('Alice Smith')).toBeInTheDocument();
    expect(screen.getByText('notRecorded')).toBeInTheDocument();
  });

  it('renders status badge when attendance exists', () => {
    renderWithProviders(
      <AttendanceTable
        rows={[rowWithAttendance]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    expect(screen.getByText('Bob Jones')).toBeInTheDocument();
    expect(screen.getByText('present')).toBeInTheDocument();
  });

  it('renders note text', () => {
    renderWithProviders(
      <AttendanceTable
        rows={[rowWithAttendance]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    expect(screen.getByText('Arrived early')).toBeInTheDocument();
  });

  it('calls onQuickStatus when quick button is clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <AttendanceTable
        rows={[baseRow]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    // Click the first quick button (present)
    const presentBtn = screen.getByRole('button', { name: 'present' });
    await user.click(presentBtn);
    expect(mockOnQuickStatus).toHaveBeenCalledWith(1, 'present', undefined);
  });

  it('calls onEdit when edit button is clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <AttendanceTable
        rows={[rowWithAttendance]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    const editBtn = screen.getByRole('button', { name: 'edit' });
    await user.click(editBtn);
    expect(mockOnEdit).toHaveBeenCalledWith(rowWithAttendance);
  });

  it('calls onDelete when delete button is clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <AttendanceTable
        rows={[rowWithAttendance]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    const deleteBtn = screen.getByRole('button', { name: 'delete' });
    await user.click(deleteBtn);
    expect(mockOnDelete).toHaveBeenCalledWith(rowWithAttendance);
  });

  it('does not render edit/delete buttons when no attendance', () => {
    renderWithProviders(
      <AttendanceTable
        rows={[baseRow]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    expect(screen.queryByRole('button', { name: 'edit' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'delete' })).not.toBeInTheDocument();
  });

  it('renders multiple rows', () => {
    renderWithProviders(
      <AttendanceTable
        rows={[baseRow, rowWithAttendance]}
        onQuickStatus={mockOnQuickStatus}
        onEdit={mockOnEdit}
        onDelete={mockOnDelete}
      />
    );
    expect(screen.getByText('Alice Smith')).toBeInTheDocument();
    expect(screen.getByText('Bob Jones')).toBeInTheDocument();
  });
});
