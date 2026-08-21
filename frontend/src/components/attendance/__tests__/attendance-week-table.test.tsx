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
    contracts: [],
    vouchers: [],
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
    contracts: [],
    vouchers: [],
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
        child_name: 'Alice Smith',
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

  /**
   * Enrolment is per day, not per week.
   *
   * The page used to fetch the roster for the Monday alone and use it for all
   * five columns, so a mid-week start had no row at all and a mid-week end kept
   * an inviting check-in button for days the contract no longer covered.
   */
  describe('per-day enrolment', () => {
    const enrolled = (perDay: Record<string, number[]>) =>
      new Map(Object.entries(perDay).map(([day, ids]) => [day, new Set(ids)]));

    function renderWeek(enrolledByDate?: Map<string, Set<number>>, attendance = new Map()) {
      return renderWithProviders(
        <AttendanceWeekTable
          childRecords={mockChildren}
          attendanceByDate={attendance}
          enrolledByDate={enrolledByDate}
          onCheckIn={noopFn}
          onCheckOut={noopFn}
          onUpdateTime={noopFn}
          onSetStatus={noopFn}
          onSaveNote={noopFn}
          days={days}
        />
      );
    }

    it('marks the days a child has no contract for', () => {
      // Bob (id 2) starts on the Tuesday.
      renderWeek(enrolled({ '2024-01-15': [1], '2024-01-16': [1, 2] }));

      expect(screen.getAllByText('notEnrolled')).toHaveLength(1);
    });

    it('still offers check-in on the days the child is enrolled', () => {
      renderWeek(enrolled({ '2024-01-15': [1], '2024-01-16': [1, 2] }));

      // Alice both days, Bob the Tuesday only: three live cells.
      expect(screen.getAllByLabelText('checkIn')).toHaveLength(3);
    });

    it('leaves every cell alone while a day is still loading', () => {
      // The Tuesday has not arrived yet, so it is absent from the map entirely.
      renderWeek(enrolled({ '2024-01-15': [1, 2] }));

      expect(screen.queryByText('notEnrolled')).not.toBeInTheDocument();
    });

    it('behaves as before when no enrolment is supplied at all', () => {
      renderWeek(undefined);

      expect(screen.queryByText('notEnrolled')).not.toBeInTheDocument();
      expect(screen.getAllByLabelText('checkIn')).toHaveLength(4);
    });

    it('shows an existing record even on a day the roster does not list', () => {
      // Data already recorded must never be hidden by a roster query -- a
      // contract corrected after the fact would erase it from view.
      const attendance = new Map<string, ChildAttendanceResponse[]>();
      attendance.set('2024-01-15', [
        {
          id: 99,
          child_id: 2,
          organization_id: 1,
          child_name: 'Bob Jones',
          date: '2024-01-15',
          status: 'sick',
          check_in_time: null,
          check_out_time: null,
          note: '',
          created_at: '2024-01-15T08:00:00Z',
          updated_at: '2024-01-15T08:00:00Z',
        },
      ]);
      renderWeek(enrolled({ '2024-01-15': [1], '2024-01-16': [1, 2] }), attendance);

      expect(screen.getByText('sick')).toBeInTheDocument();
      expect(screen.queryByText('notEnrolled')).not.toBeInTheDocument();
    });
  });
});
