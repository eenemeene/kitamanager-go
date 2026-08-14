import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useForm } from 'react-hook-form';
import { validationTiming } from '../validation-timing';

/**
 * These assert the behaviour the timing produces, not the constant's value.
 * Asserting `mode === 'onTouched'` would pass even if react-hook-form changed
 * what that means; a user only ever experiences the timing.
 */
function Harness() {
  const {
    register,
    formState: { errors },
  } = useForm<{ email: string }>({
    ...validationTiming,
    // A rule that fails on partial input, which is the whole point: "a@b" is
    // invalid on the way to a valid address.
    defaultValues: { email: '' },
  });

  return (
    <form>
      <label htmlFor="email">Email</label>
      <input
        id="email"
        {...register('email', {
          required: 'Email is required',
          pattern: { value: /^[^@\s]+@[^@\s]+\.[^@\s]+$/, message: 'Enter a valid email address' },
        })}
      />
      <button type="button">elsewhere</button>
      {errors.email && <p role="alert">{errors.email.message}</p>}
    </form>
  );
}

describe('validationTiming', () => {
  it('says nothing while the user is still typing', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.type(screen.getByLabelText('Email'), 'a');

    // The failure mode this replaces: with mode 'onChange' the user is told the
    // address is invalid after one character, before they have had a chance to
    // be right.
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('reports the problem once the user leaves the field', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.type(screen.getByLabelText('Email'), 'not-an-email');
    await user.click(screen.getByRole('button', { name: 'elsewhere' }));

    // And it does report — the other failure mode is react-hook-form's own
    // default, which stays silent until the whole form is submitted.
    expect(await screen.findByRole('alert')).toHaveTextContent('Enter a valid email address');
  });

  it('clears the error as the user fixes it, without waiting for another submit', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    const field = screen.getByLabelText('Email');
    await user.type(field, 'not-an-email');
    await user.click(screen.getByRole('button', { name: 'elsewhere' }));
    expect(await screen.findByRole('alert')).toBeInTheDocument();

    await user.clear(field);
    await user.type(field, 'someone@example.org');

    // Re-validate early: once a field has errored, it is judged on every
    // keystroke so the message disappears the moment the input becomes valid.
    await waitFor(() => expect(screen.queryByRole('alert')).not.toBeInTheDocument());
  });
});
