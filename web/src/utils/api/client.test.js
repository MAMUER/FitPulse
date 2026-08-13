import { describe, expect, it, vi } from 'vitest';
import { apiRequest, getAuthToken, setAuthToken } from './client';

describe('client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('gets token from localStorage', () => {
    const localStorageMock = window.localStorage;
    localStorageMock.getItem.mockReturnValueOnce('token123');
    expect(getAuthToken()).toBe('token123');
    expect(localStorageMock.getItem).toHaveBeenCalledWith('authToken');
  });

  it('returns null when no token', () => {
    const localStorageMock = window.localStorage;
    localStorageMock.getItem.mockReturnValueOnce(null);
    expect(getAuthToken()).toBeNull();
  });

  it('sets token in localStorage', () => {
    const localStorageMock = window.localStorage;
    setAuthToken('token123');
    expect(localStorageMock.setItem).toHaveBeenCalledWith(
      'authToken',
      'token123'
    );
  });

  it('removes token when setAuthToken is called with null', () => {
    const localStorageMock = window.localStorage;
    setAuthToken(null);
    expect(localStorageMock.removeItem).toHaveBeenCalledWith('authToken');
  });

  it('makes api request with token', async () => {
    const localStorageMock = window.localStorage;
    localStorageMock.getItem.mockReturnValueOnce('token123');

    const mockResponse = {
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({ data: 'test' }),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    const result = await apiRequest('/test');
    expect(result).toEqual({ data: 'test' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/test', {
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer token123',
      },
    });
  });

  it('makes api request without token', async () => {
    const localStorageMock = window.localStorage;
    localStorageMock.getItem.mockReturnValueOnce(null);

    const mockResponse = {
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({ data: 'test' }),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    const result = await apiRequest('/test');
    expect(result).toEqual({ data: 'test' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/test', {
      headers: {
        'Content-Type': 'application/json',
      },
    });
  });

  it('handles 401 response', async () => {
    const localStorageMock = window.localStorage;
    localStorageMock.getItem.mockReturnValueOnce('token123');

    const mockResponse = {
      ok: false,
      status: 401,
      headers: new Headers(),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    await expect(apiRequest('/test')).rejects.toThrow(
      'Сессия истекла. Войдите заново'
    );
    expect(localStorageMock.removeItem).toHaveBeenCalledWith('authToken');
  });

  it('handles 429 response with Retry-After', async () => {
    const mockResponse = {
      ok: false,
      status: 429,
      headers: new Headers({ 'Retry-After': '60' }),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    await expect(apiRequest('/test')).rejects.toThrow(
      'Слишком много запросов. Повторите через 60 сек.'
    );
  });

  it('handles 429 response without Retry-After', async () => {
    const mockResponse = {
      ok: false,
      status: 429,
      headers: new Headers(),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    await expect(apiRequest('/test')).rejects.toThrow(
      'Слишком много запросов. Попробуйте через минуту.'
    );
  });

  it('handles non-ok response with json error', async () => {
    const mockResponse = {
      ok: false,
      status: 400,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({ message: 'Bad request' }),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    await expect(apiRequest('/test')).rejects.toThrow('Bad request');
  });

  it('handles non-ok response with string error', async () => {
    const mockResponse = {
      ok: false,
      status: 500,
      headers: new Headers({ 'content-type': 'text/plain' }),
      text: () => Promise.resolve('Server error'),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    await expect(apiRequest('/test')).rejects.toThrow('Server error');
  });

  it('handles non-ok response with generic error', async () => {
    const mockResponse = {
      ok: false,
      status: 500,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({}),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    await expect(apiRequest('/test')).rejects.toThrow('Ошибка сервера (500)');
  });

  it('handles text response', async () => {
    const mockResponse = {
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'text/plain' }),
      text: () => Promise.resolve('plain text'),
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockResponse);

    const result = await apiRequest('/test');
    expect(result).toBe('plain text');
  });
});
