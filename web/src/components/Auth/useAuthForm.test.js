import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthForm } from './useAuthForm';

const mockLogin = vi.fn();
const mockRegister = vi.fn();
const mockVerify2FA = vi.fn();

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    login: mockLogin,
  }),
}));

vi.mock('../../utils/api', () => ({
  register: () => mockRegister(),
  verify2FA: () => mockVerify2FA(),
}));

describe('useAuthForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('initializes with empty form', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
        onSuccessMessage: vi.fn(),
      })
    );
    expect(result.current.formData.email).toBe('');
    expect(result.current.formData.password).toBe('');
  });

  it('sets field value', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
        onSuccessMessage: vi.fn(),
      })
    );
    act(() => {
      result.current.setField('email', 'test@test.com');
    });
    expect(result.current.formData.email).toBe('test@test.com');
  });

  it('clears error when setting field', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
        onSuccessMessage: vi.fn(),
      })
    );
    act(() => {
      result.current.setField('email', 'test@test.com');
    });
    expect(result.current.errors.email).toBe('');
  });

  it('updates password checks', () => {
    const { result } = renderHook(() =>
      useAuthForm({
        searchParams: new URLSearchParams(),
        onModeChange: vi.fn(),
        onSuccessMessage: vi.fn(),
      })
    );
    act(() => {
      result.current.updatePasswordChecks('Test1234');
    });
    expect(result.current.passwordChecks.length).toBe(true);
    expect(result.current.passwordChecks.upper).toBe(true);
    expect(result.current.passwordChecks.digit).toBe(true);
  });
});
