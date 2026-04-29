import { fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test-utils';
import { PropertyFormDialog } from '../property-form-dialog';

// PropertyFormDialog drives `governmentFundingPropertySchema`. Pinned
// behaviour:
//   - label, key, value all required (z.string().min(1))
//   - payment_euros / requirement: number >= 0
//   - min_age, max_age: optional + nullable (empty input → null)
//
// We can't reliably trigger a Radix Dialog Portal'd <Button type="submit">
// from userEvent.click in jsdom — see period-form-dialog.test.tsx for
// the same pattern. Submit by firing on the form ancestor.

function submitFormVia(input: HTMLElement) {
  const form = input.closest('form');
  if (!form) throw new Error('expected form ancestor');
  fireEvent.submit(form);
}

function renderDialog(props: Partial<React.ComponentProps<typeof PropertyFormDialog>> = {}) {
  const onSubmit: jest.Mock = jest.fn();
  const onOpenChange: jest.Mock = jest.fn();
  renderWithProviders(
    <PropertyFormDialog
      open={props.open ?? true}
      onOpenChange={onOpenChange}
      onSubmit={onSubmit}
      isSaving={props.isSaving ?? false}
    />
  );
  return { onSubmit, onOpenChange };
}

describe('PropertyFormDialog', () => {
  beforeEach(() => jest.clearAllMocks());

  describe('render', () => {
    it('renders all expected fields with correct defaults', () => {
      renderDialog();
      expect(screen.getByLabelText('governmentFundings.label')).toHaveValue('');
      expect(screen.getByLabelText('governmentFundings.key')).toHaveValue('');
      expect(screen.getByLabelText('governmentFundings.value')).toHaveValue('');
      expect(screen.getByLabelText('governmentFundings.paymentInEuros')).toHaveValue(0);
      expect(screen.getByLabelText('governmentFundings.requirement')).toHaveValue(0);
      expect(screen.getByLabelText('common.comment')).toHaveValue('');
    });

    it('does not render dialog content when closed', () => {
      renderDialog({ open: false });
      expect(screen.queryByText('governmentFundings.addProperty')).not.toBeInTheDocument();
    });
  });

  describe('happy-path submission', () => {
    it('submits a fully filled property', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      await user.type(screen.getByLabelText('governmentFundings.label'), 'Full-Time');
      await user.type(screen.getByLabelText('governmentFundings.key'), 'care_type');
      await user.type(screen.getByLabelText('governmentFundings.value'), 'ganztag');
      const payment = screen.getByLabelText('governmentFundings.paymentInEuros');
      await user.clear(payment);
      await user.type(payment, '1668.47');
      const requirement = screen.getByLabelText('governmentFundings.requirement');
      await user.clear(requirement);
      await user.type(requirement, '0.261');
      submitFormVia(screen.getByLabelText('governmentFundings.label'));
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
        label: 'Full-Time',
        key: 'care_type',
        value: 'ganztag',
        payment_euros: 1668.47,
        requirement: 0.261,
      });
    });

    it('submits min_age / max_age as numbers when provided', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      await user.type(screen.getByLabelText('governmentFundings.label'), 'U3');
      await user.type(screen.getByLabelText('governmentFundings.key'), 'care_type');
      await user.type(screen.getByLabelText('governmentFundings.value'), 'ganztag');
      const minAge = screen.getByLabelText('governmentFundings.minAge');
      const maxAge = screen.getByLabelText('governmentFundings.maxAge');
      await user.type(minAge, '0');
      await user.type(maxAge, '3');
      submitFormVia(minAge);
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
        min_age: 0,
        max_age: 3,
      });
    });

    it('leaves min_age/max_age as default null when the user does not touch them', async () => {
      // Property defaults to min_age=null, max_age=null when the user
      // omits them — meaning "applies to all ages". This is the safest
      // default; a regression to 0/0 (only newborns) would silently
      // narrow funding scope.
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      await user.type(screen.getByLabelText('governmentFundings.label'), 'All-Ages');
      await user.type(screen.getByLabelText('governmentFundings.key'), 'care_type');
      await user.type(screen.getByLabelText('governmentFundings.value'), 'any');
      submitFormVia(screen.getByLabelText('governmentFundings.label'));
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      const data = onSubmit.mock.calls[0]?.[0];
      expect(data.min_age).toBeNull();
      expect(data.max_age).toBeNull();
    });
  });

  describe('schema validation rejection', () => {
    // Each rejection test confirms onSubmit is NOT called when the
    // schema's contract is violated. Without these tests, a
    // refactor that loosens `min(1)` to `optional()` would silently
    // let empty rows through to the backend.

    it('rejects when label is empty', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const labelInput = screen.getByLabelText('governmentFundings.label');
      // label intentionally empty
      await user.type(screen.getByLabelText('governmentFundings.key'), 'k');
      await user.type(screen.getByLabelText('governmentFundings.value'), 'v');
      submitFormVia(labelInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('rejects when key is empty', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const labelInput = screen.getByLabelText('governmentFundings.label');
      await user.type(labelInput, 'L');
      // key intentionally empty
      await user.type(screen.getByLabelText('governmentFundings.value'), 'v');
      submitFormVia(labelInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('rejects when value is empty', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const labelInput = screen.getByLabelText('governmentFundings.label');
      await user.type(labelInput, 'L');
      await user.type(screen.getByLabelText('governmentFundings.key'), 'k');
      // value intentionally empty
      submitFormVia(labelInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('rejects when payment_euros is negative', async () => {
      // schema: z.number().min(0). A negative payment doesn't make
      // sense and would produce a corrupt funding period.
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const labelInput = screen.getByLabelText('governmentFundings.label');
      await user.type(labelInput, 'L');
      await user.type(screen.getByLabelText('governmentFundings.key'), 'k');
      await user.type(screen.getByLabelText('governmentFundings.value'), 'v');
      const payment = screen.getByLabelText('governmentFundings.paymentInEuros');
      await user.clear(payment);
      await user.type(payment, '-1');
      submitFormVia(labelInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('accepts payment_euros=0 (boundary)', async () => {
      // schema: min(0) inclusive. A zero-payment property is legal —
      // it documents that a category exists but doesn't pay (e.g. for
      // requirement-only entries).
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      await user.type(screen.getByLabelText('governmentFundings.label'), 'Z');
      await user.type(screen.getByLabelText('governmentFundings.key'), 'k');
      await user.type(screen.getByLabelText('governmentFundings.value'), 'v');
      // payment_euros stays at default 0
      submitFormVia(screen.getByLabelText('governmentFundings.label'));
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      expect(onSubmit.mock.calls[0]?.[0].payment_euros).toBe(0);
    });

    it('rejects when label exceeds 255 chars', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const labelInput = screen.getByLabelText('governmentFundings.label');
      await user.click(labelInput);
      await user.paste('x'.repeat(256));
      await user.type(screen.getByLabelText('governmentFundings.key'), 'k');
      await user.type(screen.getByLabelText('governmentFundings.value'), 'v');
      submitFormVia(labelInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('rejects when key exceeds 100 chars', async () => {
      const user = userEvent.setup();
      const { onSubmit } = renderDialog();
      const labelInput = screen.getByLabelText('governmentFundings.label');
      const keyInput = screen.getByLabelText('governmentFundings.key');
      await user.type(labelInput, 'L');
      await user.click(keyInput);
      await user.paste('k'.repeat(101));
      await user.type(screen.getByLabelText('governmentFundings.value'), 'v');
      submitFormVia(labelInput);
      await new Promise((r) => setTimeout(r, 50));
      expect(onSubmit).not.toHaveBeenCalled();
    });
  });

  describe('cancel behaviour', () => {
    it('calls onOpenChange(false) when cancel is clicked, without firing onSubmit', async () => {
      const user = userEvent.setup();
      const { onSubmit, onOpenChange } = renderDialog();
      await user.type(screen.getByLabelText('governmentFundings.label'), 'partial');
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
