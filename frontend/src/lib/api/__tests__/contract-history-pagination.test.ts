/**
 * Contract history is fetched in full, not one page of it.
 *
 * Both contract-list endpoints are paginated and default to 20 rows, while the
 * pages that consume them render the whole history with no pager. Reading only
 * the first page therefore dropped contract 21 onwards — and because the backend
 * orders by `from_date DESC`, what disappeared was the *oldest* history, from
 * the table and the ContractTimeline alike. Every amendment adds a row, so a
 * long-tenured child or employee reaches 20.
 *
 * Nothing else catches this: the truncated response is a valid, successful
 * response, so it fails quietly.
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

/**
 * A paginated endpoint holding `total` contracts, served in pages of whatever
 * `limit` the client asks for. Records the pages actually requested.
 */
function paginatedContracts(url: string, total: number) {
  const requestedPages: number[] = [];
  const handler = http.get(url, ({ request }) => {
    const params = new URL(request.url).searchParams;
    const limit = Number(params.get('limit'));
    const page = Number(params.get('page'));
    requestedPages.push(page);

    const start = (page - 1) * limit;
    const data = Array.from({ length: Math.max(0, Math.min(limit, total - start)) }, (_, i) => ({
      id: start + i + 1,
    }));

    return HttpResponse.json({
      data,
      page,
      limit,
      total,
      total_pages: Math.ceil(total / limit),
    });
  });
  return { handler, requestedPages };
}

describe('contract history pagination', () => {
  it('returns every child contract, not just the first page', async () => {
    // 250 rows over a page size of 100: three pages, and a count no single
    // request can return — the backend caps limit at 100.
    const { handler, requestedPages } = paginatedContracts(
      `${API_BASE}/organizations/1/children/2/contracts`,
      250
    );
    server.use(handler);

    const client = await createFreshClient();
    const contracts = await client.getChildContracts(1, 2);

    expect(contracts).toHaveLength(250);
    expect(requestedPages).toEqual([1, 2, 3]);
    // Order is preserved across the page boundary, so `from_date DESC` still
    // holds for the merged list.
    expect(contracts.map((c) => (c as { id: number }).id).slice(0, 3)).toEqual([1, 2, 3]);
  });

  it('returns every employee contract, not just the first page', async () => {
    const { handler, requestedPages } = paginatedContracts(
      `${API_BASE}/organizations/1/employees/2/contracts`,
      250
    );
    server.use(handler);

    const client = await createFreshClient();
    const contracts = await client.getEmployeeContracts(1, 2);

    expect(contracts).toHaveLength(250);
    expect(requestedPages).toEqual([1, 2, 3]);
  });

  it('makes a single request when the history fits in one page', async () => {
    const { handler, requestedPages } = paginatedContracts(
      `${API_BASE}/organizations/1/children/2/contracts`,
      3
    );
    server.use(handler);

    const client = await createFreshClient();
    const contracts = await client.getChildContracts(1, 2);

    expect(contracts).toHaveLength(3);
    expect(requestedPages).toEqual([1]);
  });

  it('makes a single request and returns nothing for an empty history', async () => {
    // total_pages is 0 here, which must not read as "keep paging".
    const { handler, requestedPages } = paginatedContracts(
      `${API_BASE}/organizations/1/children/2/contracts`,
      0
    );
    server.use(handler);

    const client = await createFreshClient();
    const contracts = await client.getChildContracts(1, 2);

    expect(contracts).toEqual([]);
    expect(requestedPages).toEqual([1]);
  });
});
