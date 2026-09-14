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
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('logs in and sets token', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ access_token: 'token123', user: { id: 1 } })
    );

    const result = await api.login('test@test.com', 'password123');
    expect(result).toEqual({ access_token: 'token123', user: { id: 1 } });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/login',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@test.com',
          password: 'password123',
        }),
      })
    );
    expect(localStorage.getItem('authToken')).toBe('token123');
  });

  it('logs in without setting token when access_token is missing', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ user: { id: 1 } })
    );

    const result = await api.login('test@test.com', 'password123');
    expect(result).toEqual({ user: { id: 1 } });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(localStorage.getItem('authToken')).toBeNull();
  });

  it('registers user', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: 1 }));

    const result = await api.register(
      'test@test.com',
      'password123',
      'Test User'
    );
    expect(result).toEqual({ id: 1 });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/register',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@test.com',
          password: 'password123',
          full_name: 'Test User',
          role: 'client',
        }),
      })
    );
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
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/register/invite',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          invite_code: 'CODE123',
          full_name: 'Test User',
          email: 'test@test.com',
          password: 'password123',
        }),
      })
    );
  });

  it('validates invite', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ valid: true })
    );

    const result = await api.validateInvite('CODE123');
    expect(result).toEqual({ valid: true });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/invite/validate',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: 'CODE123' }),
      })
    );
  });

  it('handles logout successfully', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    await api.logout();
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/logout',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      })
    );
    expect(localStorage.getItem('authToken')).toBeNull();
  });

  it('clears token on logout error', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'));
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    await api.logout();
    expect(consoleSpy).toHaveBeenCalledWith(
      'Logout request failed, clearing token anyway:',
      expect.any(Error)
    );
    expect(localStorage.getItem('authToken')).toBeNull();
    consoleSpy.mockRestore();
  });

  it('confirms email', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.confirmEmail('token123');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/auth/confirm-email',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: 'token123' }),
      })
    );
  });

  it('gets 2fa status', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ enabled: true })
    );

    const result = await api.get2FAStatus();
    expect(result).toEqual({ enabled: true });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/auth/2fa/status',
      expect.objectContaining({
        headers: { 'Content-Type': 'application/json' },
      })
    );
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
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/auth/2fa/setup',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      })
    );
  });

  it('confirms 2fa', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.confirm2FA('123456', 'secret123', ['code1']);
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/auth/2fa/confirm',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          passcode: '123456',
          temp_secret: 'secret123',
          backup_codes: ['code1'],
        }),
      })
    );
  });

  it('verifies 2fa', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ access_token: 'real' })
    );

    const result = await api.verify2FA('temp123', '123456', false);
    expect(result).toEqual({ access_token: 'real' });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/auth/2fa/verify',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          temp_token: 'temp123',
          passcode: '123456',
          is_backup_code: false,
        }),
      })
    );
  });

  it('disables 2fa', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.disable2FA('123456');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/auth/2fa/disable',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ passcode: '123456' }),
      })
    );
  });
});
