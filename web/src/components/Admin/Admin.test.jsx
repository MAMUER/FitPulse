import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import Admin from './Admin';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderAdmin = (authOverrides = {}) => {
  useAuth.mockReturnValue({
    isAdmin: true,
    ...authOverrides,
  });

  return render(
    <AuthProvider>
      <Admin />
    </AuthProvider>
  );
};

describe('Admin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows access denied for non-admin', () => {
    renderAdmin({ isAdmin: false });

    expect(screen.getByText('Доступ запрещён')).toBeInTheDocument();
  });

  it('shows loading state initially', () => {
    vi.spyOn(api, 'listInvites').mockImplementation(
      () => new Promise(() => {})
    );
    vi.spyOn(api, 'listUsers').mockImplementation(() => new Promise(() => {}));
    renderAdmin();

    expect(screen.getByText('Загрузка...')).toBeInTheDocument();
  });

  it('loads and displays empty state', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('Нет приглашений')).toBeInTheDocument();
    });

    expect(screen.getByText('Нет пользователей')).toBeInTheDocument();
  });

  it('displays invites', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([
      {
        invite_id: '1',
        code: 'ABC123',
        role: 'client',
        max_uses: 1,
        used_count: 0,
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('ABC123')).toBeInTheDocument();
    });

    expect(screen.getByText('Активно')).toBeInTheDocument();
  });

  it('displays revoked invites', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([
      {
        invite_id: '1',
        code: 'ABC123',
        role: 'client',
        max_uses: 1,
        used_count: 0,
        is_active: false,
      },
    ]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('ABC123')).toBeInTheDocument();
    });

    expect(screen.getByText('Отозвано')).toBeInTheDocument();
  });

  it('displays users', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([
      {
        user_id: '1',
        full_name: 'Test User',
        email: 'test@test.com',
        role: 'client',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
    ]);
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('Test User')).toBeInTheDocument();
    });

    expect(screen.getByText('test@test.com')).toBeInTheDocument();
  });

  it('creates invite successfully', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createInvite').mockResolvedValueOnce({});
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('Создать приглашение')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Создать'));

    expect(api.createInvite).toHaveBeenCalledWith('client', '', 1);
  });

  it('creates admin invite', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createInvite').mockResolvedValueOnce({});
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('Создать приглашение')).toBeInTheDocument();
    });

    const roleSelect = screen.getByLabelText('Роль');
    await userEvent.selectOptions(roleSelect, 'admin');

    await userEvent.click(screen.getByText('Создать'));

    expect(api.createInvite).toHaveBeenCalledWith('admin', '', 1);
  });

  it('revokes invite successfully', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([
      {
        invite_id: '1',
        code: 'ABC123',
        role: 'client',
        max_uses: 1,
        used_count: 0,
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    vi.spyOn(api, 'revokeInvite').mockResolvedValueOnce({});
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('ABC123')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Отозвать'));

    expect(api.revokeInvite).toHaveBeenCalledWith('ABC123');
  });

  it('cancels revoke when confirm is false', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([
      {
        invite_id: '1',
        code: 'ABC123',
        role: 'client',
        max_uses: 1,
        used_count: 0,
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('ABC123')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Отозвать'));

    expect(api.revokeInvite).not.toHaveBeenCalled();
  });

  it('copies invite link to clipboard', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([
      {
        invite_id: '1',
        code: 'ABC123',
        role: 'client',
        max_uses: 1,
        used_count: 0,
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('ABC123')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Скопировать ссылку'));

    expect(navigator.clipboard.writeText).toHaveBeenCalled();
    expect(window.alert).toHaveBeenCalledWith('Ссылка скопирована');
  });

  it('handles create invite error', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createInvite').mockRejectedValueOnce(
      new Error('create failed')
    );
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('Создать приглашение')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Создать'));

    expect(window.alert).toHaveBeenCalledWith('Ошибка: create failed');
  });

  it('handles load admin data error', async () => {
    vi.spyOn(api, 'listInvites').mockRejectedValueOnce(
      new Error('load failed')
    );
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('Загрузка...')).toBeInTheDocument();
    });
  });

  it('handles revoke invite error', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([
      {
        invite_id: '1',
        code: 'ABC123',
        role: 'client',
        max_uses: 1,
        used_count: 0,
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    vi.spyOn(api, 'revokeInvite').mockRejectedValueOnce(
      new Error('revoke failed')
    );
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('ABC123')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Отозвать'));

    expect(window.alert).toHaveBeenCalledWith('Ошибка: revoke failed');
  });

  it('handles clipboard copy failure', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([
      {
        invite_id: '1',
        code: 'ABC123',
        role: 'client',
        max_uses: 1,
        used_count: 0,
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockRejectedValueOnce(new Error('clipboard failed')),
      },
    });
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('ABC123')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Скопировать ссылку'));

    expect(window.alert).toHaveBeenCalledWith('Не удалось скопировать ссылку');
  });

  it('changes max uses input', async () => {
    vi.spyOn(api, 'listInvites').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listUsers').mockResolvedValueOnce([]);
    renderAdmin();

    await waitFor(() => {
      expect(screen.getByText('Создать приглашение')).toBeInTheDocument();
    });

    const maxUsesInput = screen.getByLabelText('Максимум использований');
    await userEvent.clear(maxUsesInput);
    await userEvent.type(maxUsesInput, '5');

    expect(maxUsesInput).toHaveValue(5);
  });
});
