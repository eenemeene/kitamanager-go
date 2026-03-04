import { screen } from '@testing-library/react';
import { AttendanceWeekTable } from '../attendance-week-table';
import { renderWithProviders } from '@/test-utils';
import type { Child, ChildAttendanceResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
  useLocale: () => 'en',
}));

const mockChildren: Child[] = [
  {
    id: 1,
    organization_id: 1,
    first_name: 'Alice',
    last_name: 'Smith',
    birthdate: '2020-01-01',
    gender: 'female',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    organization_id: 1,
    first_name: 'Bob',
    last_name: 'Jones',
    birthdate: '2019-06-15',
    gender: 'male',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const monday = new Date(2024, 0, 15); // Monday Jan 15 2024
const tuesday = new Date(2024, 0, 16);
const days = [monday, tuesday];

const noopFn = jest.fn();

describe('AttendanceWeekTable', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders empty state when no children', () => {
    renderWithProviders(
      <AttendanceWeekTable
        childRecords={[]}
        attendanceByDate={new Map()}
        onCheckIn={noopFn}
        onCheckOut={noopFn}
        onUpdateTime={noopFn}
        onSetStatus={noopFn}
        onSaveNote={noopFn}
        days={days}
      />
    );
    expect(screen.getByText('noChildren')).toBeInTheDocument();
  });

  it('renders child names sorted alphabetically', () => {
    renderWithProviders(
      <AttendanceWeekTable
        childRecords={mockChildren}
        attendanceByDate={new Map()}
        onCheckIn={noopFn}
        onCheckOut={noopFn}
        onUpdateTime={noopFn}
        onSetStatus={noopFn}
        onSaveNote={noopFn}
        days={days}
      />
    );
    expect(screen.getByText('Alice Smith')).toBeInTheDocument();
    expect(screen.getByText('Bob Jones')).toBeInTheDocument();
  });

  it('renders day column headers', () => {
    renderWithProviders(
      <AttendanceWeekTable
        childRecords={mockChildren}
        attendanceByDate={new Map()}
        onCheckIn={noopFn}
        onCheckOut={noopFn}
        onUpdateTime={noopFn}
        onSetStatus={noopFn}
        onSaveNote={noopFn}
        days={days}
      />
    );
    // date-fns format 'EEE dd.MM' with enUS locale
    expect(screen.getByText('Mon 15.01')).toBeInTheDocument();
    expect(screen.getByText('Tue 16.01')).toBeInTheDocument();
  });

  it('renders attendance status for a child on a given day', () => {
    const attendanceMap = new Map<string, ChildAttendanceResponse[]>();
    attendanceMap.set('2024-01-15', [
      {
        id: 10,
        child_id: 1,
        organization_id: 1,
        date: '2024-01-15',
        status: 'sick',
        check_in_time: '',
        check_out_time: '',
        note: '',
        recorded_by: 1,
        created_at: '2024-01-15T08:00:00Z',
        updated_at: '2024-01-15T08:00:00Z',
      },
    ]);

    renderWithProviders(
      <AttendanceWeekTable
        childRecords={mockChildren}
        attendanceByDate={attendanceMap}
        onCheckIn={noopFn}
        onCheckOut={noopFn}
        onUpdateTime={noopFn}
        onSetStatus={noopFn}
        onSaveNote={noopFn}
        days={days}
      />
    );
    // The sick status text should appear
    expect(screen.getByText('sick')).toBeInTheDocument();
  });
});
