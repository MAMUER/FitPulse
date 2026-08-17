import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import AuthScreen from './AuthScreen';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

vi.mock('./useAuthForm', async () => {
  const actual = await vi.importActual('./useAuthForm');
  const mockUseAuthForm = vi.fn((options) => {
    const result = actual.useAuthForm(options);
    return { ...result, generalError: 'Ошибка подтверждения email' };
  });
  return { ...actual, useAuthForm: mockUseAuthForm };
});

describe('AuthScreen generalError rendering', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuth.mockReturnValue({ login: vi.fn() });
  });

  it('renders general error in verify mode', async () => {
    render(
      <MemoryRouter>
        <AuthProvider>
          <AuthScreen searchParams={new URLSearchParams('?token=abc')} />
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Проверьте почту')).toBeInTheDocument();
    });

    expect(screen.getByText('Ошибка подтверждения email')).toBeInTheDocument();
  });
});
