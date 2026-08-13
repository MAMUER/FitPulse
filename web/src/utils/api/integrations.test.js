import { describe, expect, it, vi } from 'vitest';
import * as api from './integrations';

describe('integrations api', () => {
  const mockResponse = (data) => ({
    ok: true,
    status: 200,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve(data),
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('gets providers', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ providers: [] })
    );

    const result = await api.getProviders();
    expect(result).toEqual({ providers: [] });
    expect(fetch).toHaveBeenCalledWith('/api/v1/integrations/providers', {
      headers: { 'Content-Type': 'application/json' },
    });
  });

  it('gets providers returns apiRequest result', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ providers: [{ source: 'google' }] })
    );

    const result = await api.getProviders();
    expect(result.providers[0].source).toBe('google');
  });

  it('disconnects integration', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.disconnectIntegration('google');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/integrations/google/disconnect',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      }
    );
  });

  it('disconnects integration returns apiRequest result', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.disconnectIntegration('fitbit');
    expect(result.success).toBe(true);
  });

  it('handles api error', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: false,
      status: 500,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({ message: 'Server error' }),
    });

    await expect(api.getProviders()).rejects.toThrow('Server error');
  });
});
