import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Register from './Register';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('../components/AuthLayout', () => ({
  default: ({ children }) => <div>{children}</div>,
}));

describe('Register', () => {
  it('navigates to legal and logs in', () => {
    const doRegister = vi.fn();
    const socialLogin = vi.fn();
    const go = vi.fn();
    const openLegal = vi.fn();
    useApp.mockReturnValue({
      t: (ru, _en) => ru,
      doRegister,
      socialLogin,
      go,
      openLegal,
    });

    render(<Register />);
    fireEvent.click(screen.getByText('Соглашение'));
    expect(openLegal).toHaveBeenCalledWith('terms');
    fireEvent.click(screen.getByText('Персональные данные'));
    expect(openLegal).toHaveBeenCalledWith('consent');
    fireEvent.click(screen.getByText('Политика'));
    expect(openLegal).toHaveBeenCalledWith('privacy');
    fireEvent.click(screen.getByText('Зарегистрироваться'));
    expect(doRegister).toHaveBeenCalled();
    fireEvent.click(screen.getByText('Войти'));
    expect(go).toHaveBeenCalledWith('login');
    fireEvent.click(screen.getByText(/Яндекс/));
    expect(socialLogin).toHaveBeenCalledWith('yandex');
  });
});
