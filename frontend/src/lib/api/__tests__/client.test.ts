// We store mock functions on a shared object so that jest.mock factory
// (which is hoisted above all const declarations) can reference them.
// Using 'var' avoids the temporal dead zone that 'const'/'let' would cause.

var __mockGet: jest.Mock;
var __mockPost: jest.Mock;
var __mockPut: jest.Mock;
var __mockDel: jest.Mock;

jest.mock('axios', () => {
  __mockGet = jest.fn();
  __mockPost = jest.fn();
  __mockPut = jest.fn();
  __mockDel = jest.fn();

  const instance = {
    get: __mockGet,
    post: __mockPost,
    put: __mockPut,
    delete: __mockDel,
    interceptors: {
      request: { use: jest.fn() },
      response: { use: jest.fn() },
    },
  };
  return {
    __esModule: true,
    default: {
      create: jest.fn(() => instance),
    },
  };
});

import { getErrorMessage, apiClient } from '../client';
import { getInvalidParams, getRequestId } from '../problem';

describe('getErrorMessage', () => {
  it('shows the problem detail, which is the specific one, for an English reader', () => {
    const error = {
      response: {
        data: {
          type: 'https://kitamanager.dev/docs/reference/api/errors/#unauthorized',
          title: 'Authentication required',
          status: 401,
          detail: 'Invalid credentials',
          code: 'unauthorized',
        },
      },
    };

    expect(getErrorMessage(error, 'Fallback message')).toBe('Invalid credentials');
  });

  it('falls back to the English detail for a code with no translation', () => {
    document.cookie = 'locale=de';
    const error = {
      response: {
        data: { status: 400, detail: 'something specific went wrong', code: 'brand_new_code' },
      },
    };

    // An untranslated sentence beats a blank one, and it means adding a code on
    // the backend cannot produce an empty toast.
    expect(getErrorMessage(error, 'Fallback message')).toBe('something specific went wrong');
    document.cookie = 'locale=en';
  });

  it('exposes the rejected fields so a form can mark its inputs', () => {
    const error = {
      response: {
        data: {
          status: 400,
          code: 'validation_error',
          detail: 'weekly_hours is required',
          invalid_params: [{ field: 'weekly_hours', reason: 'is required' }],
        },
      },
    };

    expect(getInvalidParams(error)).toEqual([{ field: 'weekly_hours', reason: 'is required' }]);
    expect(getRequestId(error)).toBeUndefined();
  });

  it('carries the request id, which is what support asks for after a 500', () => {
    const error = {
      response: {
        data: { status: 500, code: 'internal_error', request_id: '0e03dc7d-9baa' },
      },
    };

    expect(getRequestId(error)).toBe('0e03dc7d-9baa');
  });

  it('shows the localized detail the server sent, for a German reader', () => {
    const error = {
      response: {
        data: {
          status: 404,
          code: 'not_found',
          // English stays on top for logs and captured responses.
          detail: 'child 7 not found in this organization',
          localized: {
            locale: 'de',
            title: 'Ressource nicht gefunden',
            detail: 'Kind 7 wurde in dieser Organisation nicht gefunden',
          },
        },
      },
    };

    // The specifics survive translation — the 7 is still there. That is what
    // the server-side catalogue buys over a per-code lookup in the client.
    expect(getErrorMessage(error, 'Fallback message')).toBe(
      'Kind 7 wurde in dieser Organisation nicht gefunden'
    );
  });

  it('falls back to the English detail when the server sent no localized block', () => {
    // What an English reader gets: the server omits `localized` entirely rather
    // than echoing English into it.
    const error = {
      response: {
        data: { status: 404, code: 'not_found', detail: 'child 7 not found in this organization' },
      },
    };

    expect(getErrorMessage(error, 'Fallback message')).toBe(
      'child 7 not found in this organization'
    );
  });

  it('exposes the localized reason per rejected field', () => {
    const error = {
      response: {
        data: {
          status: 400,
          code: 'validation_error',
          invalid_params: [
            {
              field: 'email',
              rule: 'email',
              reason: 'must be a valid email address',
              localized_reason: 'muss eine gültige E-Mail-Adresse sein',
            },
          ],
        },
      },
    };

    // Both strings travel with the field, so a form can mark the input and show
    // the reader's language without matching two arrays by index.
    expect(getInvalidParams(error)[0]).toMatchObject({
      field: 'email',
      reason: 'must be a valid email address',
      localized_reason: 'muss eine gültige E-Mail-Adresse sein',
    });
  });

  it('carries every rejected field, not just the first', () => {
    // Both producers on the server report all failures at once: the binding
    // validator returns every failing field, and a bulk import reports every bad
    // row. Only the single-field case was covered here, so nothing proved the
    // client surfaced more than one.
    const error = {
      response: {
        data: {
          status: 400,
          code: 'validation_error',
          detail:
            'add_children[0].birthdate is required; add_children[0].contracts must contain at least one entry',
          invalid_params: [
            {
              field: 'add_children[0].birthdate',
              rule: 'required',
              reason: 'is required',
              localized_reason: 'ist erforderlich',
            },
            {
              field: 'add_children[0].contracts',
              rule: 'non_empty',
              reason: 'must contain at least one entry',
              localized_reason: 'muss mindestens einen Eintrag enthalten',
            },
          ],
          localized: {
            locale: 'de',
            detail:
              'add_children[0].birthdate ist erforderlich; add_children[0].contracts muss mindestens einen Eintrag enthalten',
          },
        },
      },
    };

    const params = getInvalidParams(error);
    expect(params).toHaveLength(2);
    // Order is the server's report order, so a form can walk it directly.
    expect(params.map((p) => p.field)).toEqual([
      'add_children[0].birthdate',
      'add_children[0].contracts',
    ]);
    expect(params.every((p) => p.localized_reason)).toBe(true);

    // And the single-string fallback still names both, so a caller that shows
    // only the message does not tell the user about one problem out of two.
    const message = getErrorMessage(error, 'Fallback message');
    expect(message).toContain('birthdate');
    expect(message).toContain('contracts');
  });

  it('returns an empty list when there are no field errors', () => {
    const error = { response: { data: { status: 404, code: 'not_found', detail: 'gone' } } };
    expect(getInvalidParams(error)).toEqual([]);
  });

  it('returns fallback for error without response', () => {
    const error = new Error('Network error');

    expect(getErrorMessage(error, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for a body that is not a problem document', () => {
    const error = {
      response: {
        data: {},
      },
    };

    expect(getErrorMessage(error, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for null error', () => {
    expect(getErrorMessage(null, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for undefined error', () => {
    expect(getErrorMessage(undefined, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for non-object error', () => {
    expect(getErrorMessage('string error', 'Fallback message')).toBe('Fallback message');
    expect(getErrorMessage(123, 'Fallback message')).toBe('Fallback message');
  });
});

describe('export URL builders', () => {
  describe('getEmployeesExportUrl', () => {
    it('builds URL without filters', () => {
      const url = apiClient.getEmployeesExportUrl(1);
      expect(url).toBe('/api/v1/organizations/1/employees/export/excel');
    });

    it('builds URL with filters', () => {
      const url = apiClient.getEmployeesExportUrl(1, {
        search: 'John',
        staff_category: 'qualified',
        active_on: '2026-02-01',
      });
      expect(url).toContain('/api/v1/organizations/1/employees/export/excel?');
      expect(url).toContain('search=John');
      expect(url).toContain('staff_category=qualified');
      expect(url).toContain('active_on=2026-02-01');
    });

    it('omits undefined and empty filters', () => {
      const url = apiClient.getEmployeesExportUrl(1, {
        search: undefined,
        staff_category: '',
        active_on: '2026-02-01',
      });
      expect(url).toContain('active_on=2026-02-01');
      expect(url).not.toContain('search');
      expect(url).not.toContain('staff_category');
    });
  });

  describe('getChildrenExportUrl', () => {
    it('builds URL without filters', () => {
      const url = apiClient.getChildrenExportUrl(1);
      expect(url).toBe('/api/v1/organizations/1/children/export/excel');
    });

    it('builds URL with filters', () => {
      const url = apiClient.getChildrenExportUrl(1, {
        search: 'Max',
        section_id: '3',
        active_on: '2026-03-01',
      });
      expect(url).toContain('/api/v1/organizations/1/children/export/excel?');
      expect(url).toContain('search=Max');
      expect(url).toContain('section_id=3');
      expect(url).toContain('active_on=2026-03-01');
    });

    it('omits undefined and empty filters', () => {
      const url = apiClient.getChildrenExportUrl(2, {
        search: undefined,
        section_id: undefined,
      });
      expect(url).toBe('/api/v1/organizations/2/children/export/excel');
    });
  });
});
