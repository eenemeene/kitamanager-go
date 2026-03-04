const mockGet = jest.fn();

jest.mock('next-intl/server', () => ({
  getRequestConfig: (fn: unknown) => fn,
}));

jest.mock('next/headers', () => ({
  cookies: jest.fn().mockResolvedValue({ get: (...args: unknown[]) => mockGet(...args) }),
}));

import { defaultLocale } from '../config';

describe('i18n request config', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('uses default locale when no cookie is set', async () => {
    mockGet.mockReturnValue(undefined);
    const configFn = (await import('../request')).default as unknown as () => Promise<{
      locale: string;
    }>;
    const result = await configFn();
    expect(result.locale).toBe(defaultLocale);
  });

  it('uses locale from cookie when valid', async () => {
    mockGet.mockReturnValue({ value: 'de' });
    const configFn = (await import('../request')).default as unknown as () => Promise<{
      locale: string;
    }>;
    const result = await configFn();
    expect(result.locale).toBe('de');
  });

  it('falls back to default locale for invalid cookie value', async () => {
    mockGet.mockReturnValue({ value: 'fr' });
    const configFn = (await import('../request')).default as unknown as () => Promise<{
      locale: string;
    }>;
    const result = await configFn();
    expect(result.locale).toBe(defaultLocale);
  });
});
