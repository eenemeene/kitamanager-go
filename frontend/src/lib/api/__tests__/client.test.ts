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

  it('translates by code for a German reader rather than showing the English detail', () => {
    document.cookie = 'locale=de';
    const error = {
      response: {
        data: {
          status: 409,
          detail: 'contract periods overlap between 2026-01-01 and 2026-03-31',
          code: 'contract_overlap',
        },
      },
    };

    // The point of the whole migration: a German user gets German, and gets it
    // from the code rather than from the server guessing a language.
    expect(getErrorMessage(error, 'Fallback message')).toContain('Vertragszeiträume');
    document.cookie = 'locale=en';
  });

  it('never tells a German reader less than an English one', () => {
    document.cookie = 'locale=de';
    const error = {
      response: {
        data: {
          status: 400,
          code: 'bad_request',
          // The specifics of a bulk import live only here. A generic German
          // sentence would drop the row and the field, which is the regression
          // this test exists to prevent.
          detail: 'add_children[3].contracts[1]: from is required',
        },
      },
    };

    const message = getErrorMessage(error, 'Fallback message');
    expect(message).toContain('add_children[3].contracts[1]');
    document.cookie = 'locale=en';
  });

  it('does not append the English detail when the translation already says it all', () => {
    document.cookie = 'locale=de';
    const error = {
      response: {
        data: {
          status: 409,
          code: 'contract_overlap',
          // The server's detail for this code is a bare sentinel with nothing
          // in it, so showing it too would be noise, not information.
          detail: 'period would overlap with existing record',
        },
      },
    };

    const message = getErrorMessage(error, 'Fallback message');
    expect(message).toContain('Vertragszeiträume');
    expect(message).not.toContain('period would overlap');
    document.cookie = 'locale=en';
  });

  it('renders rejected fields in German, as specifically as the English detail', () => {
    document.cookie = 'locale=de';
    const error = {
      response: {
        data: {
          status: 400,
          code: 'validation_error',
          detail: 'email must be a valid email address; name must be at least 2 characters',
          invalid_params: [
            { field: 'email', reason: 'must be a valid email address', rule: 'email' },
            { field: 'name', reason: 'must be at least 2 characters', rule: 'min', param: '2' },
          ],
        },
      },
    };

    const message = getErrorMessage(error, 'Fallback message');
    // Same two fields as the English sentence, in German, with the bound.
    expect(message).toBe(
      'email muss eine gültige E-Mail-Adresse sein; name muss mindestens 2 Zeichen lang sein'
    );
    document.cookie = 'locale=en';
  });

  it('keeps the Retry-After seconds in German', () => {
    document.cookie = 'locale=de';
    const error = {
      response: {
        data: { status: 429, code: 'too_many_requests', params: { seconds: '30' } },
      },
    };

    expect(getErrorMessage(error, 'Fallback message')).toContain('30 Sekunden');
    document.cookie = 'locale=en';
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
