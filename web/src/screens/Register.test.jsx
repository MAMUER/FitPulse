import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Register from './Register';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'register',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
  },
  t: (ru, _en) => ru,
  doRegister: vi.fn(),
  socialLogin: vi.fn(),
  toggleHighContrast: vi.fn(),
  toggleLanguage: vi.fn(),
  openLegal: vi.fn(),
  go: vi.fn(),
};

function renderRegister() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Register />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Register', () => {
  it('renders registration form', () => {
    renderRegister();
    expect(screen.getByText('FitPulse')).toBeInTheDocument();
    expect(screen.getByText('Зарегистрироваться')).toBeInTheDocument();
  });

  it('renders social login buttons', () => {
    renderRegister();
    expect(screen.getByText('Yandex')).toBeInTheDocument();
  });

  it('navigates to login', () => {
    renderRegister();
    screen.getByText('Войти').click();
    expect(defaultContext.go).toHaveBeenCalledWith('login');
  });
});
