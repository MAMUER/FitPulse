import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { AuthProvider, useAuth } from './contexts/AuthContext';

vi.mock('./contexts/AuthContext', async () => {
  const actual = await vi.importActual('./contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderApp = (authOverrides = {}) => {
  useAuth.mockReturnValue({
    token: null,
    loading: false,
    user: null,
    isAdmin: false,
    ...authOverrides,
  });

  return render(
    <BrowserRouter>
      <AuthProvider>
        <App />
      </AuthProvider>
    </BrowserRouter>
  );
};

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    renderApp({ loading: true });
    expect(screen.getByText('Загрузка...')).toBeInTheDocument();
  });

  it('renders auth screen when not authenticated', () => {
    renderApp();
    expect(screen.getByText(/Войти/i)).toBeInTheDocument();
  });

  it('renders layout and dashboard when authenticated', () => {
    renderApp({ token: 'test-token', loading: false });
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent(
      'Обзор'
    );
  });
});
