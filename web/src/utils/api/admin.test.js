import { describe, expect, it, vi } from 'vitest';
import * as api from './admin';

describe('admin api', () => {
  const mockResponse = (data) => ({
    ok: true,
    status: 200,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve(data),
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('lists invites', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ invites: [] })
    );

    const result = await api.listInvites(1, 10, '');
    expect(result).toEqual({ invites: [] });
    expect(fetch).toHaveBeenCalledWith('/api/v1/invites?page=1&page_size=10', {
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
  });

  it('lists invites with used filter', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ invites: [] })
    );

    const result = await api.listInvites(1, 10, true);
    expect(result).toEqual({ invites: [] });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/invites?page=1&page_size=10&used=true',
      {
        headers: { 'Content-Type': 'application/json' },
        signal: expect.any(AbortSignal),
      }
    );
  });

  it('creates invite', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ code: 'ABC123' })
    );

    const result = await api.createInvite('trainer', 'cardiology', 5);
    expect(result).toEqual({ code: 'ABC123' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/invites', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        role: 'trainer',
        specialty: 'cardiology',
        max_uses: 5,
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('revokes invite', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.revokeInvite('ABC123');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/invites/ABC123/revoke', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
  });

  it('lists users', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ users: [] })
    );

    const result = await api.listUsers(1, 10);
    expect(result).toEqual({ users: [] });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/admin/users?page=1&page_size=10',
      {
        headers: { 'Content-Type': 'application/json' },
        signal: expect.any(AbortSignal),
      }
    );
  });

  it('handles api error', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: false,
      status: 500,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({ message: 'Server error' }),
    });

    await expect(api.listInvites()).rejects.toThrow('Server error');
  });
});
