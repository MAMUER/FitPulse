import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthProvider } from '../../contexts/AuthContext';
import AuthScreen from './AuthScreen';

const mockUseAuthForm = vi.fn();
vi.mock('./useAuthForm', () => ({
  useAuthForm: () => mockUseAuthForm(),
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useSearchParams: () => [
      new URLSearchParams(),
      vi.fn(),
    ],
  };
});

describe('AuthScreen', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuthForm.mockReturnValue({
      formData: { email: '', password: '', name: '', totpCode: '', backupCode: '' },
      errors: {},
      generalError: '',
      passwordChecks: { length: false, upper: false, lower: false, digit: false },
      submitting: false,
      setField: vi.fn(),
      getFieldClass: vi.fn(() => ''),
      handleLogin: vi.fn(),
      handleRegister: vi.fn(),
      handleLogin2FA: vi.fn(),
      updatePasswordChecks: vi.fn(),
    });
  });

  it('renders login form by default', () => {
    render(
      <AuthProvider>
        <AuthScreen />
      </AuthProvider>
    );
    expect(screen.getByLabelText('Форма входа')).toBeDefined();
  });

  it('switches to register mode', async () => {
    const user = userEvent.setup();
    render(
      <AuthProvider>
        <AuthScreen />
      </AuthProvider>
    );
    await user.click(screen.getByText('Создать'));
    expect(screen.getByLabelText('Форма регистрации')).toBeDefined();
  });

  it('renders privacy and terms links', () => {
    render(
      <AuthProvider>
        <AuthScreen />
      </AuthProvider>
    );
    expect(screen.getByText('Политика конфиденциальности')).toBeDefined();
    expect(screen.getByText('Пользовательское соглашение')).toBeDefined();
  });
});
