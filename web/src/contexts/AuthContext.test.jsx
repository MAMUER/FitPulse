import { render, screen, waitFor } from '@testing-library/react';
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

  it('login sets token and admin flag for admin role', async () => {
    api.login.mockResolvedValueOnce({
      access_token: 'new-token',
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
    screen.getByRole('button').click();

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
    screen.getByRole('button').click();

    await waitFor(() => {
      expect(screen.getByTestId('token')).toHaveTextContent('null');
      expect(screen.getByTestId('user')).toHaveTextContent('null');
      expect(screen.getByTestId('isAdmin')).toHaveTextContent('false');
    });
  });
});
