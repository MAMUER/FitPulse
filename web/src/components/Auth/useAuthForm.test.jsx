import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthForm } from './useAuthForm';

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}));

vi.mock('../../utils/api', () => ({
  register: vi.fn(),
  verify2FA: vi.fn(),
}));

vi.mock('../../utils/validators', () => ({
  validateEmail: vi.fn(),
  validateLoginPassword: vi.fn(),
  validateName: vi.fn(),
  validatePassword: vi.fn(),
}));

import { useAuth } from '../../contexts/AuthContext';
import { register, verify2FA } from '../../utils/api';
import {
  validateEmail,
  validateLoginPassword,
  validateName,
  validatePassword,
} from '../../utils/validators';

describe('useAuthForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuth.mockReturnValue({ login: vi.fn() });
    validateEmail.mockReturnValue('');
    validateLoginPassword.mockReturnValue('');
    validateName.mockReturnValue('');
    validatePassword.mockReturnValue({
      error: '',
      checks: { length: true, upper: true, lower: true, digit: true },
    });
  });

  it('sets field and clears errors', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('email', 'test@test.com');
    });

    expect(result.current.formData.email).toBe('test@test.com');
  });

  it('handles login2FA successfully', async () => {
    const verify2FAMock = vi
      .fn()
      .mockResolvedValue({ access_token: 'new-token' });
    verify2FA.mockImplementation(verify2FAMock);
    const loginMock = vi.fn();
    useAuth.mockReturnValue({ login: loginMock });

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('totpCode', '123456');
    });

    await act(async () => {
      await result.current.handleLogin2FA(
        { preventDefault: vi.fn() },
        'temp-token'
      );
    });

    expect(verify2FAMock).toHaveBeenCalledWith('temp-token', '123456', false);
    expect(loginMock).toHaveBeenCalledWith('new-token');
  });

  it('handles login2FA with backup code', async () => {
    const verify2FAMock = vi
      .fn()
      .mockResolvedValue({ access_token: 'new-token' });
    verify2FA.mockImplementation(verify2FAMock);
    const loginMock = vi.fn();
    useAuth.mockReturnValue({ login: loginMock });

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('backupCode', 'backup123');
    });

    await act(async () => {
      await result.current.handleLogin2FA(
        { preventDefault: vi.fn() },
        'temp-token'
      );
    });

    expect(verify2FAMock).toHaveBeenCalledWith('temp-token', 'backup123', true);
  });

  it('handles login2FA error', async () => {
    const verify2FAMock = vi.fn().mockRejectedValue(new Error('invalid code'));
    verify2FA.mockImplementation(verify2FAMock);

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('totpCode', '000000');
    });

    await act(async () => {
      await result.current.handleLogin2FA(
        { preventDefault: vi.fn() },
        'temp-token'
      );
    });

    expect(result.current.generalError).toBe('invalid code');
  });

  it('handles login successfully', async () => {
    const loginMock = vi.fn().mockResolvedValue({});
    useAuth.mockReturnValue({ login: loginMock });

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('email', 'test@test.com');
      result.current.setField('password', 'password123');
    });

    await act(async () => {
      await result.current.handleLogin({ preventDefault: vi.fn() });
    });

    expect(loginMock).toHaveBeenCalledWith('test@test.com', 'password123');
  });

  it('handles login with 2FA requirement', async () => {
    const loginMock = vi.fn().mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temp-token',
    });
    useAuth.mockReturnValue({ login: loginMock });
    const onModeChange = vi.fn();

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange,
      })
    );

    act(() => {
      result.current.setField('email', 'test@test.com');
      result.current.setField('password', 'password123');
    });

    await act(async () => {
      await result.current.handleLogin({ preventDefault: vi.fn() });
    });

    expect(onModeChange).toHaveBeenCalledWith('login2fa');
  });

  it('handles login error', async () => {
    const loginMock = vi.fn().mockRejectedValue(new Error('network error'));
    useAuth.mockReturnValue({ login: loginMock });

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('email', 'test@test.com');
      result.current.setField('password', 'password123');
    });

    await act(async () => {
      await result.current.handleLogin({ preventDefault: vi.fn() });
    });

    expect(result.current.generalError).toBe('network error');
  });

  it('handles register error', async () => {
    const registerMock = vi.fn().mockRejectedValue(new Error('email exists'));
    register.mockImplementation(registerMock);
    useAuth.mockReturnValue({ login: vi.fn() });

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('name', 'Test User');
      result.current.setField('email', 'test@test.com');
      result.current.setField('password', 'Test1234');
    });

    await act(async () => {
      await result.current.handleRegister({ preventDefault: vi.fn() });
    });

    expect(result.current.generalError).toBe('email exists');
  });

  it('handles register validation failure', async () => {
    const registerMock = vi.fn();
    register.mockImplementation(registerMock);
    useAuth.mockReturnValue({ login: vi.fn() });
    validateName.mockReturnValueOnce('Имя обязательно');

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('email', 'test@test.com');
      result.current.setField('password', 'Test1234');
    });

    await act(async () => {
      await result.current.handleRegister({ preventDefault: vi.fn() });
    });

    expect(result.current.generalError).toBe('Проверьте введённые данные');
    expect(registerMock).not.toHaveBeenCalled();
  });

  it('handles register validation with password error', async () => {
    const registerMock = vi.fn();
    register.mockImplementation(registerMock);
    useAuth.mockReturnValue({ login: vi.fn() });
    validateName.mockReturnValueOnce('');
    validateEmail.mockReturnValueOnce('');
    validatePassword.mockReturnValueOnce({
      error: 'Пароль слишком слабый',
      checks: { length: false, upper: false, lower: false, digit: false },
    });

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('name', 'Test');
      result.current.setField('email', 'test@test.com');
      result.current.setField('password', 'weak');
    });

    await act(async () => {
      await result.current.handleRegister({ preventDefault: vi.fn() });
    });

    expect(result.current.generalError).toBe('Проверьте введённые данные');
    expect(result.current.errors.password).toBe('Пароль слишком слабый');
    expect(registerMock).not.toHaveBeenCalled();
  });

  it('handles register validation with email and password errors', async () => {
    const registerMock = vi.fn();
    register.mockImplementation(registerMock);
    useAuth.mockReturnValue({ login: vi.fn() });
    validateName.mockReturnValueOnce('');
    validateEmail.mockReturnValueOnce('Некорректный email');
    validatePassword.mockReturnValueOnce({
      error: 'Пароль слишком слабый',
      checks: { length: false, upper: false, lower: false, digit: false },
    });

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setField('name', 'Test');
      result.current.setField('email', 'test@test.com');
      result.current.setField('password', 'Test1234');
    });

    await act(async () => {
      await result.current.handleRegister({ preventDefault: vi.fn() });
    });

    expect(result.current.generalError).toBe('Проверьте введённые данные');
    expect(result.current.errors.email).toBe('Некорректный email');
    expect(result.current.errors.password).toBe('Пароль слишком слабый');
    expect(registerMock).not.toHaveBeenCalled();
  });

  it('handles register success with default message', async () => {
    const registerMock = vi.fn().mockResolvedValue({});
    register.mockImplementation(registerMock);
    useAuth.mockReturnValue({ login: vi.fn() });
    const onModeChange = vi.fn();
    const onSuccessMessage = vi.fn();

    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange,
        onSuccessMessage,
      })
    );

    act(() => {
      result.current.setField('name', 'Test User');
      result.current.setField('email', 'test@test.com');
      result.current.setField('password', 'Test1234');
    });

    await act(async () => {
      await result.current.handleRegister({ preventDefault: vi.fn() });
    });

    expect(onSuccessMessage).toHaveBeenCalledWith(
      'Регистрация успешна. Подтвердите email.'
    );
    expect(onModeChange).toHaveBeenCalledWith('verify');
  });

  it('handles login2FA with empty code', async () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    await act(async () => {
      await result.current.handleLogin2FA(
        { preventDefault: vi.fn() },
        'temp-token'
      );
    });

    expect(result.current.generalError).toBe('Введите код');
  });

  it('updates password checks', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    act(() => {
      result.current.updatePasswordChecks('Test1234');
    });

    expect(result.current.passwordChecks).toEqual({
      length: true,
      upper: true,
      lower: true,
      digit: true,
    });
  });

  it('returns initial form data', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    expect(result.current.formData).toEqual({
      email: '',
      password: '',
      name: '',
      totpCode: '',
      backupCode: '',
    });
  });

  it('returns initial errors as empty object', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    expect(result.current.errors).toEqual({});
  });

  it('returns initial generalError as empty string', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    expect(result.current.generalError).toBe('');
  });

  it('returns initial submitting as false', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
      })
    );

    expect(result.current.submitting).toBe(false);
  });
});
