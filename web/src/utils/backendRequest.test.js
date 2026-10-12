import { afterEach, describe, expect, it, vi } from 'vitest';

const BASE = 'https://api.example.com/';
const BASE_WITH_PATH = 'https://api.example.com/api/';

function jsonResponse(data, status = 200) {
  return {
    status,
    ok: status >= 200 && status < 300,
    json: async () => data,
  };
}

async function setup({ baseUrl = '', production = false } = {}) {
  vi.unstubAllEnvs();

  if (baseUrl) {
    vi.stubEnv('VITE_API_BASE_URL', baseUrl);
  }

  if (production) {
    vi.stubEnv('NODE_ENV', 'production');
  }

  const fetchMock = vi.fn();
  vi.stubGlobal('fetch', fetchMock);
  vi.resetModules();

  const mod = await import('./backendRequest');

  return { mod, fetchMock };
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe('resolveBaseUrl', () => {
  let resolveBaseUrl;

  beforeEach(async () => {
    ({ resolveBaseUrl } = (await setup()).mod);
  });

  it('returns an empty string for empty and blank values', () => {
    expect(resolveBaseUrl('')).toBe('');
    expect(resolveBaseUrl('   ')).toBe('');
  });

  it('returns an empty string for unparsable values', () => {
    expect(resolveBaseUrl('not a url')).toBe('');
  });

  it('returns an empty string for disallowed protocols', () => {
    expect(resolveBaseUrl('javascript:alert(1)')).toBe('');
    expect(resolveBaseUrl('ftp://files.example.com')).toBe('');
  });

  it('keeps origin and path without trailing slashes', () => {
    expect(resolveBaseUrl('https://api.example.com')).toBe(
      'https://api.example.com'
    );
    expect(resolveBaseUrl('https://api.example.com///')).toBe(
      'https://api.example.com'
    );
    expect(resolveBaseUrl('  https://api.example.com/api/v1/  ')).toBe(
      'https://api.example.com/api/v1'
    );
  });
});

describe('backendRequest', () => {
  it('uses relative URLs when no base URL is configured', async () => {
    const { mod, fetchMock } = await setup();
    expect(mod.BACKEND_CONFIG.baseUrl).toBe('');

    fetchMock.mockResolvedValue(jsonResponse({ name: 'Михаил' }));
    await expect(mod.backendRequest('/api/v1/profile')).resolves.toEqual({
      name: 'Михаил',
    });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/profile', {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
    });
  });

  it('ignores an invalid configured base URL', async () => {
    const { mod, fetchMock } = await setup({ baseUrl: 'javascript:alert(1)' });
    expect(mod.BACKEND_CONFIG.baseUrl).toBe('');

    fetchMock.mockResolvedValue(jsonResponse({ ok: true }));
    await mod.backendRequest('/api/v1/profile');
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/profile',
      expect.any(Object)
    );
  });

  it('returns parsed json for a successful request', async () => {
    const { mod, fetchMock } = await setup({ baseUrl: BASE });
    fetchMock.mockResolvedValue(jsonResponse({ name: 'Михаил' }));
    await expect(mod.backendRequest('/api/v1/profile')).resolves.toEqual({
      name: 'Михаил',
    });
    expect(fetchMock).toHaveBeenCalledWith(
      'https://api.example.com/api/v1/profile',
      {
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
      }
    );
  });

  it('merges options and custom headers into request init', async () => {
    const { mod, fetchMock } = await setup({ baseUrl: BASE });
    fetchMock.mockResolvedValue(jsonResponse({ ok: true }));
    await mod.backendRequest('/api/v1/profile', {
      method: 'POST',
      body: '{"a":1}',
      headers: { 'X-Test': '1' },
    });
    expect(fetchMock).toHaveBeenCalledWith(
      'https://api.example.com/api/v1/profile',
      {
        method: 'POST',
        body: '{"a":1}',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', 'X-Test': '1' },
      }
    );
  });

  it('uses origin and path of a prefixed base URL', async () => {
    const { mod, fetchMock } = await setup({ baseUrl: BASE_WITH_PATH });
    fetchMock.mockResolvedValue(jsonResponse({ ok: true }));
    await mod.backendRequest('/api/v1/profile');
    expect(fetchMock).toHaveBeenCalledWith(
      'https://api.example.com/api/api/v1/profile',
      expect.any(Object)
    );
  });

  it('returns null for 204 responses', async () => {
    const { mod, fetchMock } = await setup();
    fetchMock.mockResolvedValue({
      status: 204,
      ok: true,
      json: vi.fn().mockResolvedValue({}),
    });
    await expect(mod.backendRequest('/api/v1/logout')).resolves.toBeNull();
  });

  it('throws for non-ok responses', async () => {
    const { mod, fetchMock } = await setup();
    fetchMock.mockResolvedValue({
      status: 500,
      ok: false,
      json: async () => ({ error: 'boom' }),
    });
    await expect(mod.backendRequest('/api/v1/profile')).rejects.toThrow(
      'Backend 500'
    );
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('refreshes and retries after a 401', async () => {
    const { mod, fetchMock } = await setup({
      baseUrl: BASE,
    });
    fetchMock.mockResolvedValueOnce(jsonResponse({}, 401));
    fetchMock.mockResolvedValueOnce(jsonResponse({ refreshed: true }));
    fetchMock.mockResolvedValueOnce(jsonResponse({ retried: true }));
    await expect(mod.backendRequest('/api/v1/profile')).resolves.toEqual({
      retried: true,
    });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1][0]).toBe(
      'https://api.example.com/api/v1/auth/refresh'
    );
    expect(fetchMock.mock.calls[1][1]).toEqual({
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
    });
    expect(fetchMock.mock.calls[2][0]).toBe(
      'https://api.example.com/api/v1/profile'
    );
  });

  it('falls through to error handling when refresh returns null', async () => {
    const { mod, fetchMock } = await setup();
    fetchMock.mockResolvedValueOnce(jsonResponse({}, 401));
    fetchMock.mockResolvedValueOnce({
      status: 204,
      ok: true,
      json: vi.fn().mockResolvedValue({}),
    });
    await expect(mod.backendRequest('/api/v1/profile')).rejects.toThrow(
      'Backend 401'
    );
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('throws when the retry fails', async () => {
    const { mod, fetchMock } = await setup();
    fetchMock.mockResolvedValueOnce(jsonResponse({}, 401));
    fetchMock.mockResolvedValueOnce(jsonResponse({ refreshed: true }));
    fetchMock.mockResolvedValueOnce(jsonResponse({ error: 'forbidden' }, 403));
    await expect(mod.backendRequest('/api/v1/profile')).rejects.toThrow(
      'Backend 403'
    );
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('returns null for a 204 refresh response after retry success', async () => {
    const { mod, fetchMock } = await setup();
    fetchMock.mockResolvedValueOnce(jsonResponse({}, 401));
    fetchMock.mockResolvedValueOnce(jsonResponse({ refreshed: true }));
    fetchMock.mockResolvedValueOnce({
      status: 204,
      ok: true,
      json: vi.fn().mockResolvedValue({}),
    });
    await expect(mod.backendRequest('/api/v1/logout')).resolves.toBeNull();
  });

  it('propagates fetch failures', async () => {
    const { mod, fetchMock } = await setup();
    fetchMock.mockRejectedValue(new Error('network down'));
    await expect(mod.backendRequest('/api/v1/profile')).rejects.toThrow(
      'network down'
    );
  });
});
