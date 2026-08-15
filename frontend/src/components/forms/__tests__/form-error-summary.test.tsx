import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { FormErrorSummary } from '../form-error-summary';

// next-intl is mocked globally (jest.setup.js) so translations render as their
// keys. Item text is asserted directly because it is built from props — the
// field's label and the reason — not from the catalogue.

const errors = {
  first_name: { message: 'ist erforderlich' },
  email: { message: 'muss eine gültige E-Mail-Adresse sein' },
} as never;

const labels = { first_name: 'Vorname', email: 'E-Mail-Adresse' };

// jsdom implements neither of these; both are asserted rather than merely
// stubbed, because the arguments are the behaviour.
const scrollIntoView = jest.fn();
beforeAll(() => {
  Element.prototype.scrollIntoView = scrollIntoView;
});
beforeEach(() => {
  scrollIntoView.mockClear();
  window.matchMedia = jest.fn().mockReturnValue({ matches: false }) as never;
});

describe('FormErrorSummary', () => {
  it('renders nothing when there is nothing wrong', () => {
    render(<FormErrorSummary errors={{} as never} />);
    expect(screen.queryByTestId('form-error-summary')).not.toBeInTheDocument();
  });

  it('lists one item per problem and reports how many', () => {
    render(<FormErrorSummary errors={errors} labels={labels} />);

    expect(screen.getByTestId('form-error-summary')).toHaveAttribute('data-count', '2');
    expect(screen.getAllByRole('listitem')).toHaveLength(2);
  });

  it('names the field before the reason, so the sentence reads', () => {
    render(<FormErrorSummary errors={errors} labels={labels} />);
    // Both catalogues phrase reasons as predicates, so label + reason is a
    // grammatical sentence rather than "first_name: ist erforderlich".
    expect(screen.getByRole('button', { name: 'Vorname ist erforderlich' })).toBeInTheDocument();
  });

  it('falls back to the field name when no label is supplied', () => {
    render(<FormErrorSummary errors={errors} />);
    // Ugly, but never silent — a missing label must not hide the problem.
    expect(screen.getByRole('button', { name: /first_name/ })).toBeInTheDocument();
  });

  it('takes focus itself rather than focusing an input', () => {
    // Focusing the first field would summon the on-screen keyboard on a phone or
    // a tablet in portrait, taking ~40% of the viewport and potentially hiding
    // the field it just focused.
    render(<FormErrorSummary errors={errors} labels={labels} />);
    expect(screen.getByTestId('form-error-summary')).toHaveFocus();
  });

  it('moves focus to the field when its item is activated', async () => {
    const user = userEvent.setup();
    render(
      <>
        <FormErrorSummary errors={errors} labels={labels} />
        <input id="first_name" />
      </>
    );

    await user.click(screen.getByRole('button', { name: 'Vorname ist erforderlich' }));

    expect(screen.getByRole('textbox')).toHaveFocus();
    // Centred, not top-aligned: on a phone the keyboard occupies the lower half
    // once focus lands, and a top-aligned field ends up behind it.
    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'center' });
  });

  it('scrolls instantly when the user asks for less motion', async () => {
    window.matchMedia = jest.fn().mockReturnValue({ matches: true }) as never;
    const user = userEvent.setup();
    render(
      <>
        <FormErrorSummary errors={errors} labels={labels} />
        <input id="first_name" />
      </>
    );

    await user.click(screen.getByRole('button', { name: 'Vorname ist erforderlich' }));

    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'auto', block: 'center' });
  });

  it('shows violations that have no field, instead of dropping them', () => {
    // A bulk import reports a path no single input represents. Silence here
    // would tell the user two things are wrong and show one.
    render(
      <FormErrorSummary
        errors={{ first_name: { message: 'ist erforderlich' } } as never}
        labels={labels}
        unmapped={[
          {
            field: 'add_children[3].contracts[1].from',
            rule: 'required',
            reason: 'is required',
            localized_reason: 'ist erforderlich',
          },
        ]}
      />
    );

    expect(screen.getByTestId('form-error-summary')).toHaveAttribute('data-count', '2');
    expect(screen.getByText(/add_children\[3\]/)).toBeInTheDocument();
    // Nothing to jump to, so it is not a button.
    expect(screen.getAllByRole('button')).toHaveLength(1);
  });

  it('gives every item a 44px touch target', () => {
    render(<FormErrorSummary errors={errors} labels={labels} />);
    for (const item of screen.getAllByRole('button')) {
      expect(item.className).toMatch(/min-h-11/);
    }
  });
});
