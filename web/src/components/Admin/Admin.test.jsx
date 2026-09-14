import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Admin from './Admin';

const mockUseAuth = vi.fn();
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}));

const mockCreateInvite = vi.fn();
const mockListInvites = vi.fn();
const mockListUsers = vi.fn();
const mockRevokeInvite = vi.fn();

vi.mock('../../utils/api', () => ({
  createInvite: (...args) => mockCreateInvite(...args),
  listInvites: (...args) => mockListInvites(...args),
  listUsers: (...args) => mockListUsers(...args),
  revokeInvite: (...args) => mockRevokeInvite(...args),
}));

describe('Admin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({ isAdmin: true });
  });

  it('renders admin panel for admin user', async () => {
    mockListInvites.mockResolvedValue([]);
    mockListUsers.mockResolvedValue([]);
    render(<Admin />);
    expect(await screen.findByText('Создать приглашение')).toBeDefined();
  });

  it('renders access denied for non-admin', () => {
    mockUseAuth.mockReturnValue({ isAdmin: false });
    render(<Admin />);
    expect(screen.getByText('Доступ запрещён')).toBeDefined();
  });

  it('loads invites and users', async () => {
    mockListInvites.mockResolvedValue([
      { invite_id: '1', code: 'ABC123', is_active: true, role: 'client' },
    ]);
    mockListUsers.mockResolvedValue([
      {
        user_id: '1',
        full_name: 'Test',
        email: 'test@test.com',
        role: 'client',
      },
    ]);
    render(<Admin />);
    expect(await screen.findByText('ABC123')).toBeDefined();
    expect(screen.getByText('Test')).toBeDefined();
  });

  it('creates an invite', async () => {
    const user = userEvent.setup();
    mockListInvites.mockResolvedValue([]);
    mockListUsers.mockResolvedValue([]);
    mockCreateInvite.mockResolvedValue({});
    render(<Admin />);
    expect(await screen.findByText('Создать приглашение')).toBeDefined();
    const button = screen.getByText('Создать');
    await user.click(button);
    expect(mockCreateInvite).toHaveBeenCalledWith('client', '', 1);
  });

  it('copies invite link to clipboard', async () => {
    const user = userEvent.setup();
    mockListInvites.mockResolvedValue([
      { invite_id: '1', code: 'ABC123', is_active: true },
    ]);
    mockListUsers.mockResolvedValue([]);
    const writeText = vi.fn();
    Object.defineProperty(navigator, 'clipipboard', {
      value: { writeText },
      writable: true,
      configurable: true,
    });
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      writable: true,
      configurable: true,
    });
    render(<Admin />);
    expect(await screen.findByText('Скопировать ссылку')).toBeDefined();
    await user.click(screen.getByText('Скопировать ссылку'));
    expect(writeText).toHaveBeenCalled();
  });

  it('shows no invites message when empty', async () => {
    mockListInvites.mockResolvedValue([]);
    mockListUsers.mockResolvedValue([]);
    render(<Admin />);
    expect(await screen.findByText('Нет приглашений')).toBeDefined();
  });
});
