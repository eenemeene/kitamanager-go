/**
 * @jest-environment node
 */
import { GET } from './route';

describe('GET /version', () => {
  const origEnv = process.env;

  afterEach(() => {
    process.env = origEnv;
  });

  it('returns version triple from env vars', async () => {
    process.env = {
      ...origEnv,
      APP_VERSION: 'v0.27.1',
      APP_COMMIT: 'abc123def',
      APP_BUILD_TIME: '2026-04-27T10:00:00Z',
    };
    const res = await GET();
    const body = await res.json();
    expect(body).toEqual({
      version: 'v0.27.1',
      commit: 'abc123def',
      build_time: '2026-04-27T10:00:00Z',
    });
  });

  it('falls back to safe defaults when env is unset', async () => {
    process.env = { ...origEnv };
    delete process.env.APP_VERSION;
    delete process.env.APP_COMMIT;
    delete process.env.APP_BUILD_TIME;
    const res = await GET();
    const body = await res.json();
    expect(body).toEqual({
      version: 'dev',
      commit: 'unknown',
      build_time: 'unknown',
    });
  });
});
