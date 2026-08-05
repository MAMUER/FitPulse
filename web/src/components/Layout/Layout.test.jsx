import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuth } from '../../contexts/AuthContext';
import Layout from './Layout';

vi.mock('../../contexts/AuthContext', () => ({
  ...vi.importActual('../../contexts/AuthContext'),
  useAuth: vi.fn(),
}));

const renderLayout = (initialRoute = '/', authOverrides = {}) => {
  const mockUseAuth = useAuth;
  mockUseAuth.mockReturnValue({
    token: 'test-token',
    user: { id: '1', email: 'test@test.com', role: 'user' },
    loading: false,
    isAdmin: false,
    login: vi.fn(),
    logout: vi.fn(),
    ...authOverrides,
  });

  return render(
    <MemoryRouter initialEntries={[initialRoute]}>
      <Routes>
        <Route path='/' element={<Layout />}>
          <Route index element={<div>Dashboard</div>} />
          <Route path='profile' element={<div>Profile</div>} />
          <Route path='admin' element={<div>Admin</div>} />
          <Route path='*' element={<div>Not Found</div>} />
        </Route>
      </Routes>
    </MemoryRouter>
  );
};

describe('Layout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders dashboard title on home route', () => {
    renderLayout('/');
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent(
      'Обзор'
    );
  });

  it('renders profile title on profile route', () => {
    renderLayout('/profile');
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent(
      'Профиль'
    );
  });

  it('renders admin title on admin route', () => {
    renderLayout('/admin');
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent(
      'Админка'
    );
  });

  it('renders default title on unknown route', () => {
    renderLayout('/unknown');
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent(
      'FitPulse'
    );
  });

  it('shows logout button', () => {
    renderLayout('/');
    expect(screen.getByLabelText('Выйти')).toBeInTheDocument();
  });

  it('shows admin tab when user is admin', () => {
    renderLayout('/', { isAdmin: true });
    expect(screen.getByText('Админка')).toBeInTheDocument();
  });

  it('hides admin tab when user is not admin', () => {
    renderLayout('/', { isAdmin: false });
    expect(screen.queryByText('Админка')).not.toBeInTheDocument();
  });
});
