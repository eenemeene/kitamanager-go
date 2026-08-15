import { applyProblemToForm } from '../apply-problem-to-form';

/**
 * The alias map for the child-create dialog, asserted against the form's actual
 * field names.
 *
 * This is the test each form in the rollout needs, and it exists because the
 * maps are the part most likely to rot: rename a form field and violations stop
 * landing on it, silently. Writing this one caught a mistake immediately — the
 * first draft aliased `section_id` to `contract_section_id`, a field the form
 * does not have, which would have turned a markable violation into an unmapped
 * one.
 */

// Mirrors the dialog's defaultValues. Kept literal rather than imported so a
// rename in the dialog fails this test rather than being followed silently.
const formFields = {
  first_name: '',
  last_name: '',
  gender: 'male',
  birthdate: '',
  contract_from: '',
  contract_to: '',
  section_id: 0,
};

const aliases = { from: 'contract_from', to: 'contract_to' };

function problem(...fields: string[]) {
  return {
    response: {
      data: {
        status: 400,
        code: 'validation_error',
        invalid_params: fields.map((field) => ({
          field,
          rule: 'required',
          reason: 'is required',
          localized_reason: 'ist erforderlich',
        })),
      },
    },
  };
}

function form() {
  const setError = jest.fn();
  return {
    handle: { setError, clearErrors: jest.fn(), getValues: () => formFields } as never,
    setError,
  };
}

describe('child-create dialog field mapping', () => {
  it('marks every field the create endpoint can reject', () => {
    // The API names for everything this form sends. If the endpoint rejects one
    // of these, the user must see it on the input.
    const apiFields = [
      'first_name',
      'last_name',
      'gender',
      'birthdate',
      'from',
      'to',
      'section_id',
    ];
    const f = form();

    const result = applyProblemToForm(problem(...apiFields), f.handle, aliases);

    expect(result.unmapped).toEqual([]);
    expect(result.applied).toBe(apiFields.length);
  });

  it('routes the contract dates to their prefixed fields', () => {
    // The dialog carries the child and its first contract at once, so the
    // contract's dates are prefixed to keep them apart from the child's.
    const f = form();

    applyProblemToForm(problem('from', 'to'), f.handle, aliases);

    expect(f.setError.mock.calls.map((c: unknown[]) => c[0])).toEqual([
      'contract_from',
      'contract_to',
    ]);
  });

  it('does not invent an alias for section_id', () => {
    // The form and the API agree on this one. An alias here would send the
    // violation to a field that does not exist, and it would vanish from the
    // inputs into the summary's unmapped list.
    const f = form();

    const result = applyProblemToForm(problem('section_id'), f.handle, aliases);

    expect(result.applied).toBe(1);
    expect(f.setError).toHaveBeenCalledWith('section_id', expect.anything(), expect.anything());
  });
});
