import { applyProblemToForm } from '../apply-problem-to-form';

function problem(...params: { field: string; reason: string; localized_reason?: string }[]) {
  return {
    response: {
      data: {
        status: 400,
        code: 'validation_error',
        invalid_params: params.map((p) => ({ rule: 'required', ...p })),
      },
    },
  };
}

function fakeForm(values: Record<string, unknown>) {
  const setError = jest.fn();
  const clearErrors = jest.fn();
  return {
    handle: { setError, clearErrors, getValues: () => values } as never,
    setError,
    clearErrors,
  };
}

describe('applyProblemToForm', () => {
  it('marks the field the server named, with the localized reason', () => {
    const f = fakeForm({ first_name: '', last_name: 'x' });

    const result = applyProblemToForm(
      problem({ field: 'first_name', reason: 'is required', localized_reason: 'ist erforderlich' }),
      f.handle
    );

    expect(result).toEqual({ applied: 1, unmapped: [] });
    expect(f.setError).toHaveBeenCalledWith(
      'first_name',
      { type: 'server', message: 'ist erforderlich' },
      { shouldFocus: true }
    );
  });

  it('falls back to the English reason when the server sent no localized one', () => {
    const f = fakeForm({ email: '' });

    applyProblemToForm(
      problem({ field: 'email', reason: 'must be a valid email address' }),
      f.handle
    );

    expect(f.setError).toHaveBeenCalledWith(
      'email',
      { type: 'server', message: 'must be a valid email address' },
      expect.anything()
    );
  });

  it('returns violations the form has no field for, rather than losing them', () => {
    // This is the invariant the helper exists for. setError on an unknown name
    // is a silent no-op in react-hook-form: without this the message would
    // simply disappear and the form would look like it accepted the value.
    const f = fakeForm({ first_name: '' });

    const result = applyProblemToForm(
      problem(
        { field: 'first_name', reason: 'is required' },
        { field: 'add_children[3].contracts[1].from', reason: 'is required' },
        { field: 'something_the_form_never_collects', reason: 'is invalid' }
      ),
      f.handle
    );

    expect(result.applied).toBe(1);
    expect(result.unmapped.map((p) => p.field)).toEqual([
      'add_children[3].contracts[1].from',
      'something_the_form_never_collects',
    ]);
    // Every violation is accounted for: marked or handed back, never dropped.
    expect(result.applied + result.unmapped.length).toBe(3);
  });

  it('resolves a form field whose name differs from the API field', () => {
    // Money is entered in euros and sent in cents; the contract date pair is
    // prefixed because the form carries two of them.
    const f = fakeForm({ entry_amount_euros: '', contract_from: '' });

    const result = applyProblemToForm(
      problem(
        { field: 'amount_cents', reason: 'must be >= 0' },
        { field: 'from', reason: 'is required' }
      ),
      f.handle,
      { amount_cents: 'entry_amount_euros', from: 'contract_from' }
    );

    expect(result).toEqual({ applied: 2, unmapped: [] });
    // Only the first violation carries the focus option, so match on the name
    // and message and leave the third argument to the focus test below.
    expect(f.setError.mock.calls.map((c: unknown[]) => c[0])).toEqual([
      'entry_amount_euros',
      'contract_from',
    ]);
  });

  it('focuses only the first field, so the form scrolls once', () => {
    const f = fakeForm({ a: '', b: '' });

    applyProblemToForm(
      problem({ field: 'a', reason: 'is required' }, { field: 'b', reason: 'is required' }),
      f.handle
    );

    expect(f.setError).toHaveBeenNthCalledWith(1, 'a', expect.anything(), { shouldFocus: true });
    expect(f.setError).toHaveBeenNthCalledWith(2, 'b', expect.anything(), undefined);
  });

  it('clears errors from the previous attempt before applying new ones', () => {
    const f = fakeForm({ first_name: '' });

    applyProblemToForm(problem({ field: 'first_name', reason: 'is required' }), f.handle);

    expect(f.clearErrors).toHaveBeenCalled();
  });

  it('resolves nested and indexed paths', () => {
    const f = fakeForm({ properties: [{ name: '' }] });

    const result = applyProblemToForm(
      problem({ field: 'properties.0.name', reason: 'is required' }),
      f.handle
    );

    expect(result.applied).toBe(1);
    expect(f.setError).toHaveBeenCalledWith(
      'properties.0.name',
      expect.anything(),
      expect.anything()
    );
  });

  it('does nothing at all for an error carrying no field violations', () => {
    const f = fakeForm({ first_name: '' });

    const result = applyProblemToForm(
      { response: { data: { status: 409, code: 'conflict', detail: 'nope' } } },
      f.handle
    );

    // Not even clearErrors: a conflict says nothing about which field is wrong,
    // so wiping the form's existing messages would lose information.
    expect(result).toEqual({ applied: 0, unmapped: [] });
    expect(f.clearErrors).not.toHaveBeenCalled();
    expect(f.setError).not.toHaveBeenCalled();
  });
});
