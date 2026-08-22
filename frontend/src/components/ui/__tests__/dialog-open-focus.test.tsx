import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { renderWithProviders } from '@/test-utils';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

/**
 * Where focus lands when a dialog opens.
 *
 * Radix focuses the first focusable descendant, which puts the cursor in a field
 * the user did not choose. These forms validate `onTouched` -- a field is judged
 * once the user has finished with it -- so the user's first click anywhere else
 * blurs that field and it is judged for a value they were never asked for.
 *
 * Concretely: opening Create Child and adding a property as the first action
 * answered with "First name is required" and a red "Please correct 1 entry"
 * above a form barely started.
 */
function Harness({ autoFocusFirstField }: { autoFocusFirstField?: boolean }) {
  return (
    <Dialog open>
      <DialogContent autoFocusFirstField={autoFocusFirstField}>
        <DialogHeader>
          <DialogTitle>Create Child</DialogTitle>
        </DialogHeader>
        <Input aria-label="First Name" name="first_name" />
        <button type="button">Add property</button>
      </DialogContent>
    </Dialog>
  );
}

describe('dialog focus on open', () => {
  it('focuses the dialog rather than the first field', async () => {
    renderWithProviders(<Harness />);

    await waitFor(() => {
      const input = screen.getByLabelText('First Name');
      expect(document.activeElement).not.toBe(input);
    });
    // Focus is inside the dialog, so the keyboard and screen reader are in the
    // right place -- it just is not sitting in an input.
    expect(screen.getByRole('dialog')).toHaveFocus();
  });

  it('leaves the first field untouched when the user clicks something else', async () => {
    // The reported bug in one assertion: clicking a control must not count as
    // having finished with a field the user never entered.
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    const input = screen.getByLabelText('First Name');

    await user.click(screen.getByRole('button', { name: 'Add property' }));

    expect(input).not.toHaveFocus();
    // Never focused, so never blurred, so `onTouched` has nothing to judge.
    expect(input).toHaveValue('');
  });

  it('still focuses the field where a dialog opts in', async () => {
    // The credential dialogs are one short input and the user's next act is
    // certainly to type into it.
    renderWithProviders(<Harness autoFocusFirstField />);

    await waitFor(() => expect(screen.getByLabelText('First Name')).toHaveFocus());
  });
});
