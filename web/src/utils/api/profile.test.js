import { describe, expect, it, vi } from 'vitest';
import * as api from './profile';

describe('profile api', () => {
  const mockResponse = (data) => ({
    ok: true,
    status: 200,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve(data),
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('gets profile', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ profile: {} })
    );

    const result = await api.getProfile();
    expect(result).toEqual({ profile: {} });
    expect(fetch).toHaveBeenCalledWith('/api/v1/profile', {
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
  });

  it('updates profile', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.updateProfile({ nickname: 'Test' });
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/profile', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nickname: 'Test' }),
      signal: expect.any(AbortSignal),
    });
  });

  it('changes password', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.changePassword('old123', 'new123');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/change-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        current_password: 'old123',
        new_password: 'new123',
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('changes email', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.changeEmail('new@test.com', 'password123');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/change-email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        new_email: 'new@test.com',
        password: 'password123',
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('deletes profile', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.deleteProfile('password123');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/profile', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: 'password123' }),
      signal: expect.any(AbortSignal),
    });
  });

  it('handles api error', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: false,
      status: 500,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({ message: 'Server error' }),
    });

    await expect(api.getProfile()).rejects.toThrow('Server error');
  });
});
