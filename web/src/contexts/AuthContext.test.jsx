import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../utils/api';
import { AuthProvider, useAuth } from './AuthContext';

vi.mock('../utils/api', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  register: vi.fn(),
  getProfile: vi.fn(),
}));

const renderAuth = (ui, { initialEntries = ['/'], ...options } = {}) => {
  return render(<AuthProvider>{ui}</AuthProvider>, options);
};

describe('AuthContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  const user = userEvent.setup();

  it('provides initial state when no token', () => {
    const TestComponent = () => {
      const auth = useAuth();
      return (
        <div>
          <span data-testid='token'>{auth.token ?? 'null'}</span>
          <span data-testid='loading'>{String(auth.loading)}</span>
          <span data-testid='user'>{auth.user ?? 'null'}</span>
        </div>
      );
    };

    renderAuth(<TestComponent />);
    expect(screen.getByTestId('token')).toHaveTextContent('null');
    expect(screen.getByTestId('loading')).toHaveTextContent('false');
    expect(screen.getByTestId('user')).toHaveTextContent('null');
  });

  it('sets loading false when token exists but profile fetch fails', async () => {
    api.getProfile.mockRejectedValueOnce(new Error('network error'));

    const TestComponent = () => {
      const auth = useAuth();
      return <span data-testid='loading'>{String(auth.loading)}</span>;
    };

    localStorage.setItem('authToken', 'existing-token');
    renderAuth(<TestComponent />);

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false');
    });
  });

  it('login sets isAdmin to true when apiLogin returns admin role', async () => {
    api.login.mockResolvedValueOnce({
      access_token: 'admin-token-123',
      role: 'admin',
    });

    const TestComponent = () => {
      const auth = useAuth();
      return (
        <div>
          <span data-testid='token'>{auth.token ?? 'null'}</span>
          <span data-testid='isAdmin'>{String(auth.isAdmin)}</span>
          <button
            type='button'
            onClick={() => auth.login('admin@test.com', 'pass')}
          >
            Login
          </button>
        </div>
      );
    };

    renderAuth(<TestComponent />);
    await user.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByTestId('token')).toHaveTextContent('admin-token-123');
      expect(screen.getByTestId('isAdmin')).toHaveTextContent('true');
    });
  });

  it('login sets token and admin flag for admin role', async () => {
    api.login.mockResolvedValueOnce({
      access_token: 'new-token',
      role: 'admin',
    });
    api.getProfile.mockResolvedValue({
      id: 'user-1',
      email: 'admin@test.com',
      full_name: 'Admin User',
      role: 'admin',
    });

    const TestComponent = () => {
      const auth = useAuth();
      return (
        <div>
          <span data-testid='token'>{auth.token ?? 'null'}</span>
          <span data-testid='isAdmin'>{String(auth.isAdmin)}</span>
          <button
            type='button'
            onClick={() => auth.login('admin@test.com', 'pass')}
          >
            Login
          </button>
        </div>
      );
    };

    renderAuth(<TestComponent />);
    await user.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByTestId('token')).toHaveTextContent('new-token');
      expect(screen.getByTestId('isAdmin')).toHaveTextContent('true');
    });
  });

  it('logout clears state', async () => {
    api.logout.mockResolvedValueOnce(undefined);
    localStorage.setItem('authToken', 'existing-token');

    const TestComponent = () => {
      const auth = useAuth();
      return (
        <div>
          <span data-testid='token'>{auth.token ?? 'null'}</span>
          <span data-testid='user'>{auth.user ?? 'null'}</span>
          <span data-testid='isAdmin'>{String(auth.isAdmin)}</span>
          <button type='button' onClick={() => auth.logout()}>
            Logout
          </button>
        </div>
      );
    };

    renderAuth(<TestComponent />);
    await user.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByTestId('token')).toHaveTextContent('null');
      expect(screen.getByTestId('user')).toHaveTextContent('null');
      expect(screen.getByTestId('isAdmin')).toHaveTextContent('false');
    });
  });

  it('register calls api and returns data', async () => {
    api.register.mockResolvedValueOnce({
      access_token: 'register-token',
      role: 'user',
    });

    const TestComponent = () => {
      const auth = useAuth();
      return (
        <div>
          <span data-testid='token'>{auth.token ?? 'null'}</span>
          <button
            type='button'
            onClick={() => auth.register('new@test.com', 'pass123', 'New User')}
          >
            Register
          </button>
        </div>
      );
    };

    renderAuth(<TestComponent />);
    await user.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(api.register).toHaveBeenCalledWith(
        'new@test.com',
        'pass123',
        'New User'
      );
    });
  });

  it('loads profile on mount when token exists', async () => {
    api.getProfile.mockResolvedValueOnce({
      id: 'user-1',
      email: 'test@test.com',
      full_name: 'Test User',
      role: 'user',
    });
    localStorage.setItem('authToken', 'existing-token');

    const TestComponent = () => {
      const auth = useAuth();
      return (
        <div>
          <span data-testid='loading'>{String(auth.loading)}</span>
          <span data-testid='user'>{auth.user?.full_name ?? 'null'}</span>
        </div>
      );
    };

    renderAuth(<TestComponent />);

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false');
    });
  });

  it('setToken updates token value', async () => {
    const TestComponent = () => {
      const auth = useAuth();
      return (
        <div>
          <span data-testid='token'>{auth.token ?? 'null'}</span>
          <button type='button' onClick={() => auth.setToken('direct-token')}>
            SetToken
          </button>
        </div>
      );
    };

    renderAuth(<TestComponent />);
    await user.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByTestId('token')).toHaveTextContent('direct-token');
    });
  });

  it('throws when useAuth is used outside AuthProvider', () => {
    const TestComponent = () => {
      useAuth();
      return null;
    };

    expect(() => render(<TestComponent />)).toThrow(
      'useAuth must be used within AuthProvider'
    );
  });
});
