import { render, screen } from '@testing-library/react';
import { act, fireEvent } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import AuthScreen from './AuthScreen';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderAuth = (searchParams = new URLSearchParams()) => {
  useAuth.mockReturnValue({
    login: vi.fn(),
  });

  return render(
    <MemoryRouter initialEntries={['/']}>
      <AuthProvider>
        <Routes>
          <Route path='/' element={<AuthScreen searchParams={searchParams} />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );
};

describe('AuthScreen', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders login form by default', () => {
    renderAuth();
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Пароль')).toBeInTheDocument();
    expect(screen.getByText('Войти')).toBeInTheDocument();
  });

  it('switches to register mode', async () => {
    renderAuth();
    await act(async () => {
      fireEvent.click(screen.getByText('Создать'));
    });
    expect(screen.getByPlaceholderText('Имя')).toBeInTheDocument();
    expect(screen.getByText('Создать аккаунт')).toBeInTheDocument();
  });

  it('shows validation error for empty login', async () => {
    renderAuth();
    await act(async () => {
      fireEvent.click(screen.getByText('Войти'));
    });
    expect(screen.getByText('Проверьте введённые данные')).toBeInTheDocument();
  });

  it('shows verify mode when token is in URL', () => {
    renderAuth(new URLSearchParams('?token=abc'));
    expect(screen.getByText('Проверьте почту')).toBeInTheDocument();
  });

  it('shows 2FA form when mode is login2fa', async () => {
    renderAuth();
    await act(async () => {
      useAuth.mockReturnValue({ login: vi.fn() });
    });
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
  });

  it('renders privacy and terms links', () => {
    renderAuth();
    expect(screen.getByText('Политика конфиденциальности')).toBeInTheDocument();
    expect(screen.getByText('Пользовательское соглашение')).toBeInTheDocument();
  });
});
