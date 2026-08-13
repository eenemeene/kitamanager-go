/**
 * The If-Match precondition on contract deletes.
 *
 * Deleting a contract destroys its care type, supplements and period, so the
 * server requires the version the client believes it is deleting and answers 428
 * without it. This is pinned here because nothing else catches it: every
 * server-side test sends the header by construction, and the omission only
 * surfaced as an E2E failure where the contract silently stayed on the page.
 *
 * @jest-environment node
 */
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';

const API_BASE = 'http://localhost/api/v1';

process.env.NEXT_PUBLIC_API_URL = 'http://localhost';

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

async function createFreshClient() {
  jest.resetModules();
  process.env.NEXT_PUBLIC_API_URL = 'http://localhost';
  const mod = await import('../client');
  return mod.apiClient;
}

describe('contract delete preconditions', () => {
  it('sends the contract version as a quoted If-Match header for children', async () => {
    let seen: string | null = 'not-called';
    server.use(
      http.delete(`${API_BASE}/organizations/1/children/2/contracts/7`, ({ request }) => {
        seen = request.headers.get('If-Match');
        return new HttpResponse(null, { status: 204 });
      })
    );

    const client = await createFreshClient();
    await client.deleteChildContract(1, 2, 7, 3);

    // Quoted per RFC 9110 — an unquoted value is not a valid entity tag.
    expect(seen).toBe('"3"');
  });

  it('sends the contract version as a quoted If-Match header for employees', async () => {
    let seen: string | null = 'not-called';
    server.use(
      http.delete(`${API_BASE}/organizations/1/employees/2/contracts/7`, ({ request }) => {
        seen = request.headers.get('If-Match');
        return new HttpResponse(null, { status: 204 });
      })
    );

    const client = await createFreshClient();
    await client.deleteEmployeeContract(1, 2, 7, 5);

    expect(seen).toBe('"5"');
  });

  it('surfaces a 412 so the caller can tell the user to reload', async () => {
    server.use(
      http.delete(`${API_BASE}/organizations/1/children/2/contracts/7`, () =>
        HttpResponse.json(
          { code: 'precondition_failed', message: 'changed by someone else' },
          { status: 412 }
        )
      )
    );

    const client = await createFreshClient();
    await expect(client.deleteChildContract(1, 2, 7, 1)).rejects.toMatchObject({
      response: { status: 412 },
    });
  });
});

// The date filter's parameter *name*, asserted on the wire.
//
// This is the guard that was missing: the section board's test checks that the
// client method is called with a date, and the client sent it as `contract_on` —
// a parameter the API does not declare. Gin drops unknown query parameters
// silently, so the request succeeded, the handler defaulted to today, and the
// board's date picker did nothing. Nothing failed anywhere.
describe('children list date filter', () => {
  it('sends the date as active_on, the parameter the endpoint declares', async () => {
    let seen: string | null = 'not-called';
    server.use(
      http.get(`${API_BASE}/organizations/1/children`, ({ request }) => {
        seen = new URL(request.url).searchParams.get('active_on');
        return HttpResponse.json({ data: [], page: 1, limit: 100, total: 0 });
      })
    );

    const client = await createFreshClient();
    await client.getChildrenAllForDate(1, '2026-03-01');

    expect(seen).toBe('2026-03-01');
  });

  it('sends active_on for the plain board fetch too', async () => {
    const params: URLSearchParams[] = [];
    server.use(
      http.get(`${API_BASE}/organizations/1/children`, ({ request }) => {
        params.push(new URL(request.url).searchParams);
        return HttpResponse.json({ data: [], page: 1, limit: 100, total: 0 });
      })
    );

    const client = await createFreshClient();
    await client.getChildrenAll(1);

    expect(params).toHaveLength(1);
    expect(params[0].get('active_on')).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(params[0].has('contract_on')).toBe(false);
  });
});
