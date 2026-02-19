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

describe('getErrorMessage', () => {
  it('extracts message from axios error response', () => {
    const error = {
      response: {
        data: {
          message: 'Invalid credentials',
        },
      },
    };

    expect(getErrorMessage(error, 'Fallback message')).toBe('Invalid credentials');
  });

  it('returns fallback for error without response', () => {
    const error = new Error('Network error');

    expect(getErrorMessage(error, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for error without message in response', () => {
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

describe('orgScopedCrud (groups as representative)', () => {
  beforeEach(() => {
    __mockGet.mockReset();
    __mockPost.mockReset();
    __mockPut.mockReset();
    __mockDel.mockReset();
  });

  it('getGroups(1) calls get with correct URL including page/limit params', async () => {
    const mockResponse = {
      data: { data: [], total: 0, page: 1, limit: 30, total_pages: 0 },
    };
    __mockGet.mockResolvedValue(mockResponse);

    const result = await apiClient.getGroups(1);

    expect(__mockGet).toHaveBeenCalledTimes(1);
    const url = __mockGet.mock.calls[0][0] as string;
    expect(url).toContain('/organizations/1/groups');
    expect(url).toContain('page=1');
    expect(url).toContain('limit=30');
    expect(result).toEqual(mockResponse.data);
  });

  it('getGroups(1, { search: "test" }) includes search param', async () => {
    const mockResponse = {
      data: { data: [], total: 0, page: 1, limit: 30, total_pages: 0 },
    };
    __mockGet.mockResolvedValue(mockResponse);

    await apiClient.getGroups(1, { search: 'test' });

    const url = __mockGet.mock.calls[0][0] as string;
    expect(url).toContain('search=test');
  });

  it('getGroup(1, 5) calls get with /organizations/1/groups/5', async () => {
    const mockGroup = { data: { id: 5, name: 'Group A' } };
    __mockGet.mockResolvedValue(mockGroup);

    const result = await apiClient.getGroup(1, 5);

    expect(__mockGet).toHaveBeenCalledWith('/organizations/1/groups/5');
    expect(result).toEqual(mockGroup.data);
  });

  it('createGroup(1, { name: "A" }) calls post', async () => {
    const mockCreated = { data: { id: 1, name: 'A' } };
    __mockPost.mockResolvedValue(mockCreated);

    const result = await apiClient.createGroup(1, { name: 'A' });

    expect(__mockPost).toHaveBeenCalledWith('/organizations/1/groups', { name: 'A' });
    expect(result).toEqual(mockCreated.data);
  });

  it('updateGroup(1, 5, { name: "B" }) calls put', async () => {
    const mockUpdated = { data: { id: 5, name: 'B' } };
    __mockPut.mockResolvedValue(mockUpdated);

    const result = await apiClient.updateGroup(1, 5, { name: 'B' });

    expect(__mockPut).toHaveBeenCalledWith('/organizations/1/groups/5', { name: 'B' });
    expect(result).toEqual(mockUpdated.data);
  });

  it('deleteGroup(1, 5) calls delete', async () => {
    __mockDel.mockResolvedValue({});

    await apiClient.deleteGroup(1, 5);

    expect(__mockDel).toHaveBeenCalledWith('/organizations/1/groups/5');
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
