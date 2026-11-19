import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Login from './Login';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('../components/AuthLayout', () => ({
  default: ({ children }) => <div>{children}</div>,
}));

describe('Login', () => {
  it('handles actions', () => {
    const doLogin = vi.fn();
    const continueAsGuest = vi.fn();
    const socialLogin = vi.fn();
    const forgotPassword = vi.fn();
    const go = vi.fn();
    const openLegal = vi.fn();
    useApp.mockReturnValue({
      t: (ru, _en) => ru,
      doLogin,
      continueAsGuest,
      socialLogin,
      forgotPassword,
      go,
      openLegal,
    });
    render(<Login />);
    fireEvent.click(screen.getByText('Войти'));
    expect(doLogin).toHaveBeenCalled();
    fireEvent.click(screen.getByText('Зарегистрироваться'));
    expect(go).toHaveBeenCalledWith('register');
    fireEvent.click(screen.getByText('Забыли пароль?'));
    expect(forgotPassword).toHaveBeenCalled();
    fireEvent.click(screen.getByText('Продолжить как гость'));
    expect(continueAsGuest).toHaveBeenCalled();
    fireEvent.click(screen.getByText('Пользовательское соглашение'));
    expect(openLegal).toHaveBeenCalledWith('terms');
    fireEvent.click(
      screen.getByText('Соглашение об использовании персональных данных')
    );
    expect(openLegal).toHaveBeenCalledWith('consent');
    fireEvent.click(screen.getByText('Политику конфиденциальности'));
    expect(openLegal).toHaveBeenCalledWith('privacy');
    fireEvent.click(screen.getByText(/Яндекс/));
    expect(socialLogin).toHaveBeenCalledWith('yandex');
  });
});
