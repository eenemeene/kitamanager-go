import { screen } from '@testing-library/react';
import { AttendanceEditDialog } from '../attendance-edit-dialog';
import { renderWithProviders } from '@/test-utils';
import type { ChildAttendanceResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

const mockOnOpenChange = jest.fn();
const mockOnSubmit = jest.fn();

const mockAttendance: ChildAttendanceResponse = {
  id: 1,
  child_id: 5,
  organization_id: 1,
  date: '2024-01-15',
  status: 'present',
  check_in_time: '08:30:00',
  check_out_time: '16:00:00',
  note: 'Test note',
  recorded_by: 1,
  created_at: '2024-01-15T08:00:00Z',
  updated_at: '2024-01-15T08:00:00Z',
};

describe('AttendanceEditDialog', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders child name when open', () => {
    renderWithProviders(
      <AttendanceEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        attendance={mockAttendance}
        childName="Alice Smith"
        isSaving={false}
        onSubmit={mockOnSubmit}
      />
    );
    expect(screen.getByText('Alice Smith')).toBeInTheDocument();
  });

  it('renders status options', () => {
    renderWithProviders(
      <AttendanceEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        attendance={mockAttendance}
        childName="Alice Smith"
        isSaving={false}
        onSubmit={mockOnSubmit}
      />
    );
    expect(screen.getByText('status')).toBeInTheDocument();
  });

  it('renders time input fields', () => {
    renderWithProviders(
      <AttendanceEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        attendance={mockAttendance}
        childName="Alice Smith"
        isSaving={false}
        onSubmit={mockOnSubmit}
      />
    );
    expect(screen.getByLabelText('checkIn')).toBeInTheDocument();
    expect(screen.getByLabelText('checkOut')).toBeInTheDocument();
  });

  it('renders note textarea', () => {
    renderWithProviders(
      <AttendanceEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        attendance={mockAttendance}
        childName="Alice Smith"
        isSaving={false}
        onSubmit={mockOnSubmit}
      />
    );
    expect(screen.getByLabelText('note')).toBeInTheDocument();
  });

  it('does not render content when closed', () => {
    renderWithProviders(
      <AttendanceEditDialog
        open={false}
        onOpenChange={mockOnOpenChange}
        attendance={mockAttendance}
        childName="Alice Smith"
        isSaving={false}
        onSubmit={mockOnSubmit}
      />
    );
    expect(screen.queryByText('Alice Smith')).not.toBeInTheDocument();
  });
});
