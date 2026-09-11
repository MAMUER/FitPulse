import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Layout from './Layout';

const mockUseAuth = vi.fn();
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useLocation: () => ({ pathname: '/' }),
  };
});

describe('Layout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({ logout: vi.fn(), isAdmin: false });
  });

  it('renders layout with outlet', () => {
    render(
      <MemoryRouter>
        <Layout />
      </MemoryRouter>
    );
    expect(screen.getByRole('main')).toBeDefined();
  });

  it('renders navigation tabs', () => {
    render(
      <MemoryRouter>
        <Layout />
      </MemoryRouter>
    );
    expect(screen.getByText('Обзор')).toBeDefined();
    expect(screen.getByText('Профиль')).toBeDefined();
    expect(screen.getByText('Тренировки')).toBeDefined();
  });

  it('calls logout when logout button clicked', async () => {
    const user = userEvent.setup();
    const logout = vi.fn();
    mockUseAuth.mockReturnValue({ logout, isAdmin: false });
    render(
      <MemoryRouter>
        <Layout />
      </MemoryRouter>
    );
    await user.click(screen.getByLabelText('Выйти из аккаунта'));
    expect(logout).toHaveBeenCalled();
  });

  it('shows admin tab when isAdmin is true', () => {
    mockUseAuth.mockReturnValue({ logout: vi.fn(), isAdmin: true });
    render(
      <MemoryRouter>
        <Layout />
      </MemoryRouter>
    );
    expect(screen.getByText('Админка')).toBeDefined();
  });
});
