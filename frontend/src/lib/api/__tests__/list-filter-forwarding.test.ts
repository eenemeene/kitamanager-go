/**
 * Every list endpoint must put the caller's filters on the wire.
 *
 * The top-level CRUD builder used to hand-build its query string from `page`
 * and `limit` alone, so anything else the caller passed was dropped without a
 * word. The Fördersätze page passes `search`; the backend supports it
 * (`parseSearch` in internal/handlers/government_funding.go); the query key
 * included the term so every keystroke refetched. What came back was the
 * unfiltered list — a search box that looked like it worked and matched
 * everything.
 *
 * These assert on the URL rather than on rendered output because the URL is
 * where the bug lived: the call happened, the response was well-formed, and the
 * only thing wrong was what was asked for.
 */

var __mockGet: jest.Mock;

jest.mock('axios', () => {
  __mockGet = jest.fn();
  const instance = {
    get: __mockGet,
    post: jest.fn(),
    put: jest.fn(),
    patch: jest.fn(),
    delete: jest.fn(),
    interceptors: {
      request: { use: jest.fn() },
      response: { use: jest.fn() },
    },
  };
  return { __esModule: true, default: { create: jest.fn(() => instance) } };
});

import { apiClient } from '../client';

/** The path + query string of the single GET the client issued. */
function requestedUrl(): string {
  expect(__mockGet).toHaveBeenCalledTimes(1);
  return __mockGet.mock.calls[0][0] as string;
}

beforeEach(() => {
  __mockGet.mockReset();
  __mockGet.mockResolvedValue({ data: { data: [], total: 0, page: 1, limit: 30, total_pages: 0 } });
});

describe('top-level list endpoints', () => {
  it('forwards search to the government funding rates list', async () => {
    await apiClient.getGovernmentFundings({ page: 1, search: 'Berlin' });
    const url = requestedUrl();
    expect(url).toContain('search=Berlin');
    expect(url).toContain('page=1');
  });

  it('percent-encodes a filter value rather than breaking the query string', async () => {
    // A Kita name with a space and an ampersand is ordinary; string
    // interpolation would have produced a query string with a stray separator.
    await apiClient.getGovernmentFundings({ page: 1, search: 'Sonne & Mond' });
    const url = requestedUrl();
    const query = new URLSearchParams(url.slice(url.indexOf('?') + 1));
    expect(query.get('search')).toBe('Sonne & Mond');
  });

  it('omits a filter that was not set, rather than sending an empty one', async () => {
    // `search: undefined` is what the page passes before the user types. An
    // empty `search=` would be a different request from no search at all.
    await apiClient.getGovernmentFundings({ page: 2, search: undefined });
    const url = requestedUrl();
    expect(url).not.toContain('search');
    expect(url).toContain('page=2');
  });

  it('still defaults page and limit when given nothing', async () => {
    await apiClient.getOrganizations();
    const url = requestedUrl();
    expect(url).toContain('page=1');
    expect(url).toContain('limit=30');
  });

  it('carries an explicit limit for lookup-sized fetches', async () => {
    await apiClient.getUsers({ page: 1, limit: 100 });
    expect(requestedUrl()).toContain('limit=100');
  });
});

describe('organization-scoped list endpoints', () => {
  it('forwards every filter the children list takes', async () => {
    await apiClient.getChildren(7, {
      page: 3,
      search: 'Emma',
      section_id: 2,
      active_on: '2026-03-01',
    });
    const url = requestedUrl();
    expect(url).toContain('/organizations/7/children?');
    const query = new URLSearchParams(url.slice(url.indexOf('?') + 1));
    expect(query.get('search')).toBe('Emma');
    expect(query.get('section_id')).toBe('2');
    expect(query.get('active_on')).toBe('2026-03-01');
    expect(query.get('page')).toBe('3');
  });

  it('drops an unset filter but keeps the ones that are set', async () => {
    await apiClient.getEmployees(7, { page: 1, search: undefined, staff_category: 'qualified' });
    const url = requestedUrl();
    expect(url).not.toContain('search');
    expect(url).toContain('staff_category=qualified');
  });
});
