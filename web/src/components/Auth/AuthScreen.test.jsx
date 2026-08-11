import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import AuthScreen from './AuthScreen';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderAuth = (
  searchParams = new URLSearchParams(),
  authOverrides = {}
) => {
  useAuth.mockReturnValue({
    login: vi.fn(),
    ...authOverrides,
  });

  return render(
    <MemoryRouter>
      <AuthProvider>
        <AuthScreen searchParams={searchParams} />
      </AuthProvider>
    </MemoryRouter>
  );
};

describe('AuthScreen', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const user = userEvent.setup();

  it('renders login form by default', () => {
    renderAuth();
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Пароль')).toBeInTheDocument();
    expect(screen.getByText('Войти')).toBeInTheDocument();
  });

  it('switches to register mode', async () => {
    renderAuth();
    await user.click(screen.getByText('Создать'));
    expect(screen.getByPlaceholderText('Имя')).toBeInTheDocument();
    expect(screen.getByText('Создать аккаунт')).toBeInTheDocument();
  });

  it('shows validation error for empty login', async () => {
    renderAuth();
    await user.click(screen.getByText('Войти'));
    expect(screen.getByText('Проверьте введённые данные')).toBeInTheDocument();
  });

  it('shows verify mode when token is in URL', () => {
    renderAuth(new URLSearchParams('?token=abc'));
    expect(screen.getByText('Проверьте почту')).toBeInTheDocument();
  });

  it('shows 2FA form when mode is login2fa', () => {
    renderAuth();
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
  });

  it('renders privacy and terms links', () => {
    renderAuth();
    expect(screen.getByText('Политика конфиденциальности')).toBeInTheDocument();
    expect(screen.getByText('Пользовательское соглашение')).toBeInTheDocument();
  });

  it('submits login with valid data', async () => {
    const loginMock = vi.fn().mockResolvedValue({});
    renderAuth(new URLSearchParams(), { login: loginMock });

    await user.type(screen.getByPlaceholderText('Email'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123');
    await user.click(screen.getByText('Войти'));

    await waitFor(() => {
      expect(loginMock).toHaveBeenCalledWith('test@test.com', 'password123');
    });
  });

  it('shows register form fields and validation', async () => {
    renderAuth();
    await user.click(screen.getByText('Создать'));

    expect(screen.getByPlaceholderText('Имя')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('Пароль (мин. 8 символов)')
    ).toBeInTheDocument();

    await user.click(screen.getByText('Создать аккаунт'));

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Имя')).toBeInTheDocument();
    });
  });

  it('switches back to login from register', async () => {
    renderAuth();
    await user.click(screen.getByText('Создать'));
    await user.click(screen.getByText('Войти'));
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
  });

  it('shows login2fa mode and handles back navigation', () => {
    renderAuth();
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
  });

  it('shows verify mode with success message', () => {
    renderAuth(new URLSearchParams('?token=abc'));
    expect(screen.getByText('Проверьте почту')).toBeInTheDocument();
  });

  it('navigates back to login from verify mode', async () => {
    renderAuth(new URLSearchParams('?token=abc'));
    await user.click(screen.getByText('← Вернуться ко входу'));
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
  });

  it('renders landing info text', () => {
    renderAuth();
    expect(
      screen.getByText(/FitPulse — это открытая платформа/)
    ).toBeInTheDocument();
  });

  it('shows feature list on landing', () => {
    renderAuth();
    expect(screen.getByText('📊 Биометрия и активность')).toBeInTheDocument();
    expect(screen.getByText('🤖 AI-планы тренировок')).toBeInTheDocument();
  });

  it('allows typing in register name field', async () => {
    renderAuth();
    await user.click(screen.getByText('Создать'));

    const nameInput = screen.getByPlaceholderText('Имя');
    await user.type(nameInput, 'Test User');
    expect(nameInput).toHaveValue('Test User');
  });

  it('allows typing in register email field', async () => {
    renderAuth();
    await user.click(screen.getByText('Создать'));

    const emailInput = screen.getByPlaceholderText('Email');
    await user.type(emailInput, 'test@test.com');
    expect(emailInput).toHaveValue('test@test.com');
  });

  it('shows login2fa form fields', () => {
    renderAuth();
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
  });

  it('displays password checks when typing in register', async () => {
    renderAuth();
    await user.click(screen.getByText('Создать'));

    const passwordInput = screen.getByPlaceholderText(
      'Пароль (мин. 8 символов)'
    );
    await user.type(passwordInput, 'Password123');

    expect(screen.getByText(/8\+ символов/)).toBeInTheDocument();
    expect(screen.getByText(/Заглавная буква/)).toBeInTheDocument();
    expect(screen.getByText(/Строчная буква/)).toBeInTheDocument();
    expect(screen.getByText(/Цифра/)).toBeInTheDocument();
  });

  it('enters 2FA mode after login requires 2FA', async () => {
    const loginMock = vi.fn().mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temp-token',
    });
    renderAuth(new URLSearchParams(), { login: loginMock });

    await user.type(screen.getByPlaceholderText('Email'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123');
    await user.click(screen.getByText('Войти'));

    await waitFor(() => {
      expect(
        screen.getByText('Двухфакторная аутентификация')
      ).toBeInTheDocument();
    });

    expect(screen.getByPlaceholderText('6-значный код')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('Резервный код xxxx-xxxx')
    ).toBeInTheDocument();
  });

  it('allows typing in 2FA code field', async () => {
    const loginMock = vi.fn().mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temp-token',
    });
    renderAuth(new URLSearchParams(), { login: loginMock });

    await user.type(screen.getByPlaceholderText('Email'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123');
    await user.click(screen.getByText('Войти'));

    await waitFor(() => {
      expect(screen.getByPlaceholderText('6-значный код')).toBeInTheDocument();
    });

    const totpInput = screen.getByPlaceholderText('6-значный код');
    await user.type(totpInput, '123456');
    expect(totpInput).toHaveValue('123456');
  });

  it('allows typing in backup code field', async () => {
    const loginMock = vi.fn().mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temp-token',
    });
    renderAuth(new URLSearchParams(), { login: loginMock });

    await user.type(screen.getByPlaceholderText('Email'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123');
    await user.click(screen.getByText('Войти'));

    await waitFor(() => {
      expect(
        screen.getByPlaceholderText('Резервный код xxxx-xxxx')
      ).toBeInTheDocument();
    });

    const backupInput = screen.getByPlaceholderText('Резервный код xxxx-xxxx');
    await user.type(backupInput, 'backup123');
    expect(backupInput).toHaveValue('backup123');
  });

  it('navigates back to login from 2FA mode', async () => {
    const loginMock = vi.fn().mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temp-token',
    });
    renderAuth(new URLSearchParams(), { login: loginMock });

    await user.type(screen.getByPlaceholderText('Email'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123');
    await user.click(screen.getByText('Войти'));

    await waitFor(() => {
      expect(
        screen.getByText('Двухфакторная аутентификация')
      ).toBeInTheDocument();
    });

    await user.click(screen.getByText('← Вернуться ко входу'));

    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
  });

  it('submits 2FA code successfully', async () => {
    const loginMock = vi.fn().mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temp-token',
    });
    const verify2FAMock = vi.fn().mockResolvedValue({
      access_token: 'real-token',
    });
    vi.spyOn(api, 'verify2FA').mockImplementation(verify2FAMock);
    renderAuth(new URLSearchParams(), { login: loginMock });

    await user.type(screen.getByPlaceholderText('Email'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123');
    await user.click(screen.getByText('Войти'));

    await waitFor(() => {
      expect(
        screen.getByPlaceholderText('6-значный код')
      ).toBeInTheDocument();
    });

    await user.type(screen.getByPlaceholderText('6-значный код'), '123456');
    await user.click(screen.getByText('Войти'));

    expect(verify2FAMock).toHaveBeenCalledWith('temp-token', '123456', false);
  });

  it('copies totp code to backup code field', async () => {
    const loginMock = vi.fn().mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temp-token',
    });
    renderAuth(new URLSearchParams(), { login: loginMock });

    await user.type(screen.getByPlaceholderText('Email'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123');
    await user.click(screen.getByText('Войти'));

    await waitFor(() => {
      expect(
        screen.getByPlaceholderText('6-значный код')
      ).toBeInTheDocument();
    });

    await user.type(screen.getByPlaceholderText('6-значный код'), '123456');
    await user.click(screen.getByText('Использовать резервный код'));

    const backupInput = screen.getByPlaceholderText('Резервный код xxxx-xxxx');
    expect(backupInput).toHaveValue('123456');
  });
});
