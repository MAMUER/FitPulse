import { describe, expect, it, vi } from 'vitest';
import * as api from './auth';

describe('auth api', () => {
  const mockResponse = (data) => ({
    ok: true,
    status: 200,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve(data),
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('logs in and sets token', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ access_token: 'token123', user: { id: 1 } })
    );

    const result = await api.login('test@test.com', 'password123');
    expect(result).toEqual({ access_token: 'token123', user: { id: 1 } });
    expect(fetch).toHaveBeenCalledWith('/api/v1/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'test@test.com', password: 'password123' }),
      signal: expect.any(AbortSignal),
    });
    expect(window.localStorage.setItem).toHaveBeenCalledWith(
      'authToken',
      'token123'
    );
  });

  it('logs in without setting token when access_token is missing', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ user: { id: 1 } })
    );

    const result = await api.login('test@test.com', 'password123');
    expect(result).toEqual({ user: { id: 1 } });
    expect(window.localStorage.setItem).not.toHaveBeenCalledWith(
      'authToken',
      expect.any(String)
    );
  });

  it('registers user', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: 1 }));

    const result = await api.register(
      'test@test.com',
      'password123',
      'Test User'
    );
    expect(result).toEqual({ id: 1 });
    expect(fetch).toHaveBeenCalledWith('/api/v1/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: 'test@test.com',
        password: 'password123',
        full_name: 'Test User',
        role: 'client',
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('registers with invite', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: 1 }));

    const result = await api.registerWithInvite(
      'CODE123',
      'Test User',
      'test@test.com',
      'password123'
    );
    expect(result).toEqual({ id: 1 });
    expect(fetch).toHaveBeenCalledWith('/api/v1/register/invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        invite_code: 'CODE123',
        full_name: 'Test User',
        email: 'test@test.com',
        password: 'password123',
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('validates invite', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ valid: true })
    );

    const result = await api.validateInvite('CODE123');
    expect(result).toEqual({ valid: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/invite/validate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code: 'CODE123' }),
      signal: expect.any(AbortSignal),
    });
  });

  it('handles logout successfully', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    await api.logout();
    expect(fetch).toHaveBeenCalledWith('/api/v1/logout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
    expect(window.localStorage.removeItem).toHaveBeenCalledWith('authToken');
  });

  it('clears token on logout error', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'));
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    await api.logout();
    expect(consoleSpy).toHaveBeenCalledWith(
      'Logout request failed, clearing token anyway:',
      expect.any(Error)
    );
    expect(window.localStorage.removeItem).toHaveBeenCalledWith('authToken');
    consoleSpy.mockRestore();
  });

  it('confirms email', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.confirmEmail('token123');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/confirm-email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: 'token123' }),
      signal: expect.any(AbortSignal),
    });
  });

  it('gets 2fa status', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ enabled: true })
    );

    const result = await api.get2FAStatus();
    expect(result).toEqual({ enabled: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/2fa/status', {
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
  });

  it('sets up 2fa', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        qr_code_base64: 'base64',
        secret: 'secret123',
        backup_codes: [],
      })
    );

    const result = await api.setup2FA();
    expect(result).toEqual({
      qr_code_base64: 'base64',
      secret: 'secret123',
      backup_codes: [],
    });
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/2fa/setup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
  });

  it('confirms 2fa', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.confirm2FA('123456', 'secret123', ['code1']);
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/2fa/confirm', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        passcode: '123456',
        temp_secret: 'secret123',
        backup_codes: ['code1'],
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('verifies 2fa', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ access_token: 'real' })
    );

    const result = await api.verify2FA('temp123', '123456', false);
    expect(result).toEqual({ access_token: 'real' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/2fa/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        temp_token: 'temp123',
        passcode: '123456',
        is_backup_code: false,
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('disables 2fa', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.disable2FA('123456');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/2fa/disable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ passcode: '123456' }),
      signal: expect.any(AbortSignal),
    });
  });
});
