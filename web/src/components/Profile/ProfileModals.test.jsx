import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import ChangeEmailModal from './ChangeEmailModal';
import ChangePasswordModal from './ChangePasswordModal';
import DeleteProfileModal from './DeleteProfileModal';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderModal = (ModalComponent, props = {}, authOverrides = {}) => {
  useAuth.mockReturnValue({
    token: 'test-token',
    logout: vi.fn(),
    ...authOverrides,
  });

  return render(
    <AuthProvider>
      <ModalComponent onClose={vi.fn()} {...props} />
    </AuthProvider>
  );
};

describe('ChangeEmailModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    HTMLFormElement.prototype.checkValidity = () => true;
  });

  afterEach(() => {
    delete HTMLFormElement.prototype.checkValidity;
  });

  it('renders modal with form fields', () => {
    renderModal(ChangeEmailModal);

    expect(screen.getByText('Сменить почту')).toBeInTheDocument();
    expect(screen.getByLabelText('Новый email')).toBeInTheDocument();
    expect(screen.getByLabelText('Текущий пароль')).toBeInTheDocument();
    expect(screen.getByText('Сохранить новую почту')).toBeInTheDocument();
  });

  it('shows error for empty fields', async () => {
    renderModal(ChangeEmailModal);

    const form = screen.getByLabelText('Новый email').closest('form');
    fireEvent.submit(form);

    expect(screen.getByText('Заполните все поля')).toBeInTheDocument();
  });

  it('shows error for invalid email', async () => {
    renderModal(ChangeEmailModal);

    await userEvent.type(screen.getByLabelText('Новый email'), 'invalid');
    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    const form = screen.getByLabelText('Новый email').closest('form');
    fireEvent.submit(form);

    expect(screen.getByText('Некорректный email')).toBeInTheDocument();
  });

  it('shows error for email without domain dot', async () => {
    renderModal(ChangeEmailModal);

    await userEvent.type(screen.getByLabelText('Новый email'), 'test@domain');
    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    const form = screen.getByLabelText('Новый email').closest('form');
    fireEvent.submit(form);

    expect(screen.getByText('Некорректный email')).toBeInTheDocument();
  });

  it('shows error for email with empty local part', async () => {
    renderModal(ChangeEmailModal);

    await userEvent.type(screen.getByLabelText('Новый email'), '@test.com');
    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    const form = screen.getByLabelText('Новый email').closest('form');
    fireEvent.submit(form);

    expect(screen.getByText('Некорректный email')).toBeInTheDocument();
  });

  it('submits email change successfully', async () => {
    vi.spyOn(api, 'changeEmail').mockResolvedValueOnce({});
    const onClose = vi.fn();
    renderModal(ChangeEmailModal, { onClose });

    await userEvent.type(screen.getByLabelText('Новый email'), 'new@test.com');
    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    const form = screen.getByLabelText('Новый email').closest('form');
    fireEvent.submit(form);

    expect(api.changeEmail).toHaveBeenCalledWith('new@test.com', 'password123');
    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
    });
  });

  it('handles submission error', async () => {
    vi.spyOn(api, 'changeEmail').mockRejectedValueOnce(
      new Error('change failed')
    );
    renderModal(ChangeEmailModal);

    await userEvent.type(screen.getByLabelText('Новый email'), 'new@test.com');
    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    const form = screen.getByLabelText('Новый email').closest('form');
    fireEvent.submit(form);

    await waitFor(() => {
      expect(screen.getByText('change failed')).toBeInTheDocument();
    });
  });

  it('calls onClose when cancel is clicked', async () => {
    const onClose = vi.fn();
    renderModal(ChangeEmailModal, { onClose });

    await userEvent.click(screen.getByText('Отмена'));

    expect(onClose).toHaveBeenCalled();
  });

  it('closes on Escape key // NOSONAR', async () => {
    const onClose = vi.fn();
    renderModal(ChangeEmailModal, { onClose });

    const overlay = document.querySelector('.modal-overlay');
    const event = new KeyboardEvent('keydown', {
      key: 'Escape',
      code: 'Escape',
      keyCode: 27,
      which: 27,
      bubbles: true,
    });
    overlay.dispatchEvent(event);

    expect(onClose).toHaveBeenCalled();
  });

  it('does not close on non-Escape key // NOSONAR', async () => {
    const onClose = vi.fn();
    renderModal(ChangeEmailModal, { onClose });

    const overlay = document.querySelector('.modal-overlay');
    const event = new KeyboardEvent('keydown', {
      key: 'Enter',
      code: 'Enter',
      keyCode: 13,
      which: 13,
      bubbles: true,
    });
    overlay.dispatchEvent(event);

    expect(onClose).not.toHaveBeenCalled();
  });
});

describe('ChangePasswordModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    HTMLFormElement.prototype.checkValidity = () => true;
  });

  afterEach(() => {
    delete HTMLFormElement.prototype.checkValidity;
  });

  it('renders modal with form fields', () => {
    renderModal(ChangePasswordModal);

    expect(screen.getByText('Сменить пароль')).toBeInTheDocument();
    expect(screen.getByLabelText('Текущий пароль')).toBeInTheDocument();
    expect(screen.getByLabelText('Новый пароль')).toBeInTheDocument();
    expect(screen.getByLabelText('Подтверждение пароля')).toBeInTheDocument();
  });

  it('shows error for empty fields', async () => {
    renderModal(ChangePasswordModal);

    const form = screen.getByLabelText('Текущий пароль').closest('form');
    fireEvent.submit(form);

    expect(screen.getByText('Введите текущий пароль')).toBeInTheDocument();
  });

  it('shows error for short password', async () => {
    renderModal(ChangePasswordModal);

    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    await userEvent.type(screen.getByLabelText('Новый пароль'), 'short');
    const form = screen.getByLabelText('Текущий пароль').closest('form');
    fireEvent.submit(form);

    expect(screen.getByText('Минимум 8 символов')).toBeInTheDocument();
  });

  it('shows error when passwords do not match', async () => {
    renderModal(ChangePasswordModal);

    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    await userEvent.type(
      screen.getByLabelText('Новый пароль'),
      'newpassword123'
    );
    await userEvent.type(
      screen.getByLabelText('Подтверждение пароля'),
      'different123'
    );
    const form = screen.getByLabelText('Текущий пароль').closest('form');
    fireEvent.submit(form);

    expect(screen.getByText('Пароли не совпадают')).toBeInTheDocument();
  });

  it('submits password change successfully', async () => {
    vi.spyOn(api, 'changePassword').mockResolvedValueOnce({});
    const onClose = vi.fn();
    renderModal(ChangePasswordModal, { onClose });

    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    await userEvent.type(
      screen.getByLabelText('Новый пароль'),
      'newpassword123'
    );
    await userEvent.type(
      screen.getByLabelText('Подтверждение пароля'),
      'newpassword123'
    );
    const form = screen.getByLabelText('Текущий пароль').closest('form');
    fireEvent.submit(form);

    expect(api.changePassword).toHaveBeenCalledWith(
      'password123',
      'newpassword123'
    );
    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
    });
  });

  it('handles submission error', async () => {
    vi.spyOn(api, 'changePassword').mockRejectedValueOnce(
      new Error('change failed')
    );
    renderModal(ChangePasswordModal);

    await userEvent.type(
      screen.getByLabelText('Текущий пароль'),
      'password123'
    );
    await userEvent.type(
      screen.getByLabelText('Новый пароль'),
      'newpassword123'
    );
    await userEvent.type(
      screen.getByLabelText('Подтверждение пароля'),
      'newpassword123'
    );
    const form = screen.getByLabelText('Текущий пароль').closest('form');
    fireEvent.submit(form);

    await waitFor(() => {
      expect(api.changePassword).toHaveBeenCalledWith(
        'password123',
        'newpassword123'
      );
    });
  });

  it('closes on Escape key // NOSONAR', async () => {
    const onClose = vi.fn();
    renderModal(ChangePasswordModal, { onClose });

    const overlay = document.querySelector('.modal-overlay');
    const event = new KeyboardEvent('keydown', {
      key: 'Escape',
      code: 'Escape',
      keyCode: 27,
      which: 27,
      bubbles: true,
    });
    overlay.dispatchEvent(event);

    expect(onClose).toHaveBeenCalled();
  });

  it('does not close on non-Escape key // NOSONAR', async () => {
    const onClose = vi.fn();
    renderModal(ChangePasswordModal, { onClose });

    const overlay = document.querySelector('.modal-overlay');
    const event = new KeyboardEvent('keydown', {
      key: 'Enter',
      code: 'Enter',
      keyCode: 13,
      which: 13,
      bubbles: true,
    });
    overlay.dispatchEvent(event);

    expect(onClose).not.toHaveBeenCalled();
  });
});

describe('DeleteProfileModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders modal with warning and form', () => {
    renderModal(DeleteProfileModal);

    expect(screen.getByText('Удаление аккаунта')).toBeInTheDocument();
    expect(screen.getByText(/Это действие необратимо/)).toBeInTheDocument();
    expect(
      screen.getByLabelText('Введите пароль для подтверждения')
    ).toBeInTheDocument();
    expect(screen.getByText('Удалить аккаунт')).toBeInTheDocument();
  });

  it('shows error for empty password', async () => {
    renderModal(DeleteProfileModal);

    await userEvent.click(screen.getByText('Удалить аккаунт'));

    expect(screen.getByText('Введите пароль')).toBeInTheDocument();
  });

  it('calls onClose when cancel is clicked', async () => {
    const onClose = vi.fn();
    renderModal(DeleteProfileModal, { onClose });

    await userEvent.click(screen.getByText('Отмена'));

    expect(onClose).toHaveBeenCalled();
  });

  it('deletes profile and logs out on confirmation', async () => {
    vi.spyOn(api, 'deleteProfile').mockResolvedValueOnce({});
    vi.spyOn(window, 'confirm').mockImplementation(() => true);
    const logout = vi.fn();
    renderModal(DeleteProfileModal, {}, { logout });

    await userEvent.type(
      screen.getByLabelText('Введите пароль для подтверждения'),
      'password123'
    );
    await userEvent.click(screen.getByText('Удалить аккаунт'));

    expect(api.deleteProfile).toHaveBeenCalledWith('password123');
    expect(logout).toHaveBeenCalled();
  });

  it('cancels delete when confirm is false', async () => {
    vi.spyOn(api, 'deleteProfile').mockResolvedValueOnce({});
    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    renderModal(DeleteProfileModal);

    await userEvent.type(
      screen.getByLabelText('Введите пароль для подтверждения'),
      'password123'
    );
    await userEvent.click(screen.getByText('Удалить аккаунт'));

    expect(api.deleteProfile).not.toHaveBeenCalled();
  });

  it('handles deletion error', async () => {
    vi.resetAllMocks();
    vi.spyOn(api, 'deleteProfile').mockRejectedValueOnce(
      new Error('delete failed')
    );
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    renderModal(DeleteProfileModal);

    await userEvent.type(
      screen.getByLabelText('Введите пароль для подтверждения'),
      'password123'
    );
    await userEvent.click(screen.getByText('Удалить аккаунт'));

    await waitFor(() => {
      expect(screen.getByText('delete failed')).toBeInTheDocument();
    });
    expect(api.deleteProfile).toHaveBeenCalledWith('password123');
  });

  it('closes on Escape key // NOSONAR', async () => {
    const onClose = vi.fn();
    renderModal(DeleteProfileModal, { onClose });

    const overlay = document.querySelector('.modal-overlay');
    const event = new KeyboardEvent('keydown', {
      key: 'Escape',
      code: 'Escape',
      keyCode: 27,
      which: 27,
      bubbles: true,
    });
    overlay.dispatchEvent(event);

    expect(onClose).toHaveBeenCalled();
  });

  it('does not close on non-Escape key // NOSONAR', async () => {
    const onClose = vi.fn();
    renderModal(DeleteProfileModal, { onClose });

    const overlay = document.querySelector('.modal-overlay');
    const event = new KeyboardEvent('keydown', {
      key: 'Enter',
      code: 'Enter',
      keyCode: 13,
      which: 13,
      bubbles: true,
    });
    overlay.dispatchEvent(event);

    expect(onClose).not.toHaveBeenCalled();
  });
});
