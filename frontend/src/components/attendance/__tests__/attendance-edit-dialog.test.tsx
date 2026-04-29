import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AttendanceEditDialog } from '../attendance-edit-dialog';
import { renderWithProviders } from '@/test-utils';
import type { ChildAttendanceResponse } from '@/lib/api/types';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}));

const mockOnOpenChange = jest.fn();
const mockOnSubmit = jest.fn();

// API field shape: check_in_time / check_out_time are RFC3339 datetimes.
// formatTime() in formatting.ts does `new Date(iso).getHours()` — passing
// a bare 'HH:MM:SS' string was a bug in the original fixture that made
// formatTime return '' (Invalid Date) and the dialog look unprefilled.
//
// We use local-time strings (no `Z` suffix) so `new Date(...)` parses the
// hour exactly as written, regardless of the test runner's timezone. This
// makes assertions like `toHaveValue('08:30')` deterministic on Berlin,
// UTC, and US-PST CI runners alike.
const mockAttendance: ChildAttendanceResponse = {
  id: 1,
  child_id: 5,
  child_name: 'Alice Smith',
  organization_id: 1,
  date: '2024-01-15',
  status: 'present',
  check_in_time: '2024-01-15T08:30:00',
  check_out_time: '2024-01-15T16:00:00',
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

// --- Form behaviour tests (added for Phase 6 form-coverage push) -------------
//
// Render-only tests above prove the dialog mounts; these tests exercise the
// behaviour the user actually depends on: pre-filling, submission, schema
// validation, and the saving state. Each failing test would correspond to a
// real regression a user could hit (lost data on edit, accepting invalid
// data, button stays clickable while a save is in flight).
//
// Helper for a complete fixture so individual tests stay readable.
function renderDialog(
  overrides: {
    attendance?: ChildAttendanceResponse | null;
    isSaving?: boolean;
    onSubmit?: jest.Mock;
  } = {}
) {
  const onSubmit = overrides.onSubmit ?? jest.fn();
  const onOpenChange = jest.fn();
  const attendance = overrides.attendance ?? mockAttendance;
  renderWithProviders(
    <AttendanceEditDialog
      open
      onOpenChange={onOpenChange}
      attendance={attendance}
      childName="Alice Smith"
      isSaving={overrides.isSaving ?? false}
      onSubmit={onSubmit}
    />
  );
  return { onSubmit, onOpenChange };
}

describe('AttendanceEditDialog — form behaviour', () => {
  beforeEach(() => jest.clearAllMocks());

  describe('pre-fill from attendance prop', () => {
    it('seeds time and note inputs from the existing record', () => {
      // The user sees existing values when they click "edit" — without
      // this, every edit would silently overwrite the row with blanks.
      // formatTime trims seconds, so '08:30:00' renders as '08:30'.
      renderDialog();
      expect(screen.getByLabelText('checkIn')).toHaveValue('08:30');
      expect(screen.getByLabelText('checkOut')).toHaveValue('16:00');
      expect(screen.getByLabelText('note')).toHaveValue('Test note');
    });

    it('renders empty inputs when the record has no times set', () => {
      // Edge case: a record may legitimately lack check_in/out (e.g. a
      // 'sick' or 'vacation' status). The form should not display the
      // string 'undefined' or crash on formatTime(undefined).
      // The strict generated type marks these required, but the Go
      // DTO uses *time.Time pointers — a real-world response can omit
      // them. Cast the fixture to bypass the strictness.
      renderDialog({
        attendance: {
          ...mockAttendance,
          status: 'sick',
          check_in_time: undefined as unknown as string,
          check_out_time: undefined as unknown as string,
          note: '',
        },
      });
      expect(screen.getByLabelText('checkIn')).toHaveValue('');
      expect(screen.getByLabelText('checkOut')).toHaveValue('');
      expect(screen.getByLabelText('note')).toHaveValue('');
    });

    it('handles attendance=null without crashing (defensive)', () => {
      // The reset effect short-circuits on null. The dialog still
      // mounts; default values apply.
      expect(() => renderDialog({ attendance: null })).not.toThrow();
    });
  });

  describe('happy-path submission', () => {
    it('calls onSubmit with the current form values when user saves unchanged data', async () => {
      // react-hook-form's handleSubmit passes (data, event), so we
      // inspect the first positional arg via mock.calls rather than
      // toHaveBeenCalledWith (which would also match against the event).
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      await user.click(screen.getByRole('button', { name: /save/i }));
      await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
      expect(onSubmit.mock.calls[0]?.[0]).toEqual({
        status: 'present',
        check_in_time: '08:30',
        check_out_time: '16:00',
        note: 'Test note',
      });
    });

    it('reflects edits in the submitted payload', async () => {
      // Edit each field, then submit — verify each change reaches
      // onSubmit. Catches a regression where setValue / register
      // misroutes a field (e.g. wires note input to status by accident).
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const note = screen.getByLabelText('note');
      await user.clear(note);
      await user.type(note, 'updated note');
      await user.click(screen.getByRole('button', { name: /save/i }));
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({ note: 'updated note' });
    });
  });

  describe('schema validation rejection (non-happy paths)', () => {
    it('rejects submission when note exceeds 500 chars (per attendanceSchema)', async () => {
      // attendanceSchema caps note at 500 chars. Without this, very long
      // notes would propagate to the API and either be truncated server-
      // side or rejected with a less-friendly error. The form should
      // catch this client-side.
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const note = screen.getByLabelText('note');
      await user.clear(note);
      // 501 'a' chars
      await user.click(note);
      await user.paste('a'.repeat(501));
      await user.click(screen.getByRole('button', { name: /save/i }));
      // Wait long enough for zod to reject and onSubmit NOT to fire.
      await new Promise((r) => setTimeout(r, 100));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('accepts a 500-char note (boundary)', async () => {
      // The cap is inclusive: exactly 500 must pass. A regression that
      // changed `.max(500)` to `.max(499)` would silently break common
      // long-form notes at the boundary.
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const note = screen.getByLabelText('note');
      await user.clear(note);
      await user.click(note);
      await user.paste('a'.repeat(500));
      await user.click(screen.getByRole('button', { name: /save/i }));
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    });
  });

  describe('saving state', () => {
    it('does not block submission paths beyond what CrudFormDialog handles', async () => {
      // The component itself doesn't gate the submit button — that's
      // CrudFormDialog's job via the isSaving prop. We just verify the
      // prop is forwarded by checking the submit button's disabled
      // state when isSaving=true. If a future refactor breaks the
      // forwarding (e.g. passes !isSaving by mistake), this test
      // catches it without depending on dialog internals.
      renderDialog({ isSaving: true });
      const submit = screen.getByRole('button', { name: /save/i });
      expect(submit).toBeDisabled();
    });

    it('keeps the submit button enabled when not saving', () => {
      renderDialog({ isSaving: false });
      const submit = screen.getByRole('button', { name: /save/i });
      expect(submit).not.toBeDisabled();
    });
  });
});
