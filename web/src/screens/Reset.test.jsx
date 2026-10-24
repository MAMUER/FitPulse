import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Reset from './Reset';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('Reset', () => {
  it('handles step 0', async () => {
    const submitResetEmail = vi.fn().mockResolvedValue({});
    const go = vi.fn();
    useApp.mockReturnValue({
      state: { resetStep: 0 },
      t: (ru, _en) => ru,
      submitResetEmail,
      submitResetCode: vi.fn(),
      submitNewPassword: vi.fn(),
      go,
    });
    render(<Reset />);
    fireEvent.click(screen.getByText('Получить код'));
    await waitFor(() => expect(submitResetEmail).toHaveBeenCalled());
    fireEvent.click(screen.getByText('Другие варианты входа'));
    expect(go).toHaveBeenCalledWith('login');
  });

  it('handles step 1', () => {
    const submitResetCode = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: { resetStep: 1, resetEmail: 'a@b.com' },
      t: (ru, _en) => ru,
      submitResetEmail: vi.fn(),
      submitResetCode,
      submitNewPassword: vi.fn(),
      go,
    });
    render(<Reset />);
    fireEvent.change(screen.getByLabelText('Код'), {
      target: { value: '123456' },
    });
    fireEvent.click(screen.getByText('Продолжить'));
    expect(submitResetCode).toHaveBeenCalled();
    fireEvent.click(screen.getByText('Назад'));
    expect(go).toHaveBeenCalledWith('login');
  });

  it('handles step 2', () => {
    const submitNewPassword = vi.fn();
    useApp.mockReturnValue({
      state: { resetStep: 2 },
      t: (ru, _en) => ru,
      submitResetEmail: vi.fn(),
      submitResetCode: vi.fn(),
      submitNewPassword,
      go: vi.fn(),
    });
    render(<Reset />);
    fireEvent.change(screen.getByLabelText('Новый пароль'), {
      target: { value: 'pass' },
    });
    fireEvent.change(screen.getByLabelText('Подтверждение пароля'), {
      target: { value: 'pass' },
    });
    fireEvent.click(screen.getByText('Сохранить'));
    expect(submitNewPassword).toHaveBeenCalled();
  });
});
