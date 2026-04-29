import { fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test-utils';
import { PeriodFormDialog } from '../period-form-dialog';

// userEvent.type on `<input type="date">` is unreliable in jsdom (it tries
// to type each character into the date sub-fields one at a time and
// doesn't always commit). fireEvent.change with a target.value is the
// canonical jsdom-compatible way to set a date input's value.
function setDate(input: HTMLElement, isoDate: string) {
  fireEvent.change(input, { target: { value: isoDate } });
}

// userEvent.click on a Radix Dialog Portal'd <Button type="submit"> reaches
// the button but doesn't always fire the form's submit event in jsdom.
// Walking up to the form and dispatching submit directly exercises the
// same `handleSubmit(onSubmit)` resolver path consistently.
function submitFormVia(input: HTMLElement) {
  const form = input.closest('form');
  if (!form) throw new Error('expected form ancestor');
  fireEvent.submit(form);
}

// PeriodFormDialog drives `governmentFundingPeriodSchema`. The rules
// pinned by these tests:
//   - `from` is required (z.string().min(1))
//   - `full_time_weekly_hours` clamped to (0.1, 80]
//   - the `endDateAfterStart` refine: if `to` is set, it must not be
//     before `from` — otherwise a typo could record a backwards period
//     that the backend would have to fight.
//   - `to` is optional (open-ended periods are valid)
//
// We do NOT mock the Zod schema — the resolver runs the real schema
// so a regression in either the schema OR the dialog's wiring shows
// up as a failed test rather than passing-but-broken behaviour.

function renderDialog(props: Partial<React.ComponentProps<typeof PeriodFormDialog>> = {}) {
  // Always use jest.fn() locally so the returned mock retains its
  // .mock typing. Callers wanting to substitute can do so via the
  // returned object.
  const onSubmit: jest.Mock = jest.fn();
  const onOpenChange: jest.Mock = jest.fn();
  renderWithProviders(
    <PeriodFormDialog
      open={props.open ?? true}
      onOpenChange={onOpenChange}
      onSubmit={onSubmit}
      isSaving={props.isSaving ?? false}
    />
  );
  return { onSubmit, onOpenChange };
}

describe('PeriodFormDialog', () => {
  beforeEach(() => jest.clearAllMocks());

  describe('render', () => {
    it('renders the four form fields with expected default values', () => {
      renderDialog();
      expect(screen.getByLabelText('governmentFundings.fromDate')).toHaveValue('');
      expect(screen.getByLabelText('governmentFundings.toDateOptional')).toHaveValue('');
      expect(screen.getByLabelText('governmentFundings.fullTimeWeeklyHours')).toHaveValue(39);
      expect(screen.getByLabelText('common.comment')).toHaveValue('');
    });

    it('does not render dialog content when closed', () => {
      renderDialog({ open: false });
      expect(screen.queryByText('governmentFundings.addPeriod')).not.toBeInTheDocument();
    });
  });

  describe('happy-path submission', () => {
    it('calls onSubmit with form values for an open-ended period (no `to`)', async () => {
      const { onSubmit } = renderDialog();
      const fromInput = screen.getByLabelText('governmentFundings.fromDate') as HTMLInputElement;
      setDate(fromInput, '2025-01-01');
      expect(fromInput.value).toBe('2025-01-01');
      submitFormVia(fromInput);
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
        from: '2025-01-01',
        full_time_weekly_hours: 39,
      });
      expect(onSubmit.mock.calls[0]?.[0].to).toBe('');
    });

    it('passes a closed period (from + to) through unchanged', async () => {
      const { onSubmit } = renderDialog();
      const fromInput = screen.getByLabelText('governmentFundings.fromDate');
      setDate(fromInput, '2025-01-01');
      setDate(screen.getByLabelText('governmentFundings.toDateOptional'), '2025-12-31');
      submitFormVia(fromInput);
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
        from: '2025-01-01',
        to: '2025-12-31',
      });
    });

    it('passes the comment field through', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const fromInput = screen.getByLabelText('governmentFundings.fromDate');
      setDate(fromInput, '2025-01-01');
      await user.type(screen.getByLabelText('common.comment'), 'Berlin Q1 2025 rates');
      submitFormVia(fromInput);
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      expect(onSubmit.mock.calls[0]?.[0].comment).toBe('Berlin Q1 2025 rates');
    });
  });

  describe('schema validation rejection (non-happy paths)', () => {
    // For rejection tests we submit via fireEvent.submit on the form so
    // the resolver definitely runs — otherwise the test could pass
    // trivially because the click never reached the submit pipeline at
    // all (which we proved happens with this Radix Dialog setup).
    it('rejects submission when `from` is empty', async () => {
      // Schema requires from.min(1). An open-ended period without a
      // start anchor is meaningless; the schema must catch that.
      const { onSubmit } = renderDialog();
      const fromInput = screen.getByLabelText('governmentFundings.fromDate');
      submitFormVia(fromInput);
      // Wait for the resolver to settle. If it ever DID call onSubmit
      // (regression: `from` accidentally optional), waitFor would
      // surface the failure quickly rather than racing.
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('rejects when `to` is before `from` (endDateAfterStart refine)', async () => {
      // The shared `endDateAfterStart` refine catches typos. A
      // backwards period (Dec 2025 → Jan 2025) would otherwise hit
      // the API and produce an opaque 400.
      const { onSubmit } = renderDialog();
      const fromInput = screen.getByLabelText('governmentFundings.fromDate');
      setDate(fromInput, '2025-12-01');
      setDate(screen.getByLabelText('governmentFundings.toDateOptional'), '2025-01-01');
      submitFormVia(fromInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('rejects full_time_weekly_hours below 0.1 (schema bound)', async () => {
      // Schema: z.number().min(0.1).max(80). 0 is below the min — a
      // 0-hour week is meaningless.
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const fromInput = screen.getByLabelText('governmentFundings.fromDate');
      const hoursInput = screen.getByLabelText('governmentFundings.fullTimeWeeklyHours');
      setDate(fromInput, '2025-01-01');
      await user.clear(hoursInput);
      await user.type(hoursInput, '0');
      submitFormVia(fromInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('rejects full_time_weekly_hours above 80 (no full-week >80h)', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const fromInput = screen.getByLabelText('governmentFundings.fromDate');
      const hoursInput = screen.getByLabelText('governmentFundings.fullTimeWeeklyHours');
      setDate(fromInput, '2025-01-01');
      await user.clear(hoursInput);
      await user.type(hoursInput, '81');
      submitFormVia(fromInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('rejects comment longer than 1000 chars', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const fromInput = screen.getByLabelText('governmentFundings.fromDate');
      const commentInput = screen.getByLabelText('common.comment');
      setDate(fromInput, '2025-01-01');
      await user.click(commentInput);
      await user.paste('x'.repeat(1001));
      submitFormVia(fromInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });
  });

  describe('cancel behaviour', () => {
    it('calls onOpenChange(false) when cancel is clicked, without firing onSubmit', async () => {
      const user = userEvent.setup();
      const { onSubmit, onOpenChange } = renderDialog();
      setDate(screen.getByLabelText('governmentFundings.fromDate'), '2025-01-01');
      await user.click(screen.getByRole('button', { name: 'common.cancel' }));
      expect(onOpenChange).toHaveBeenCalledWith(false);
      expect(onSubmit).not.toHaveBeenCalled();
    });
  });

  describe('saving state', () => {
    it('disables the submit button while isSaving is true', () => {
      renderDialog({ isSaving: true });
      expect(screen.getByRole('button', { name: 'common.save' })).toBeDisabled();
    });

    it('keeps the submit button enabled when not saving', () => {
      renderDialog({ isSaving: false });
      expect(screen.getByRole('button', { name: 'common.save' })).not.toBeDisabled();
    });
  });
});
