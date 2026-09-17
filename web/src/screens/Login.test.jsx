import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Login from './Login';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultState = {
  screen: 'login',
  theme: 'dark',
  highContrast: false,
  language: 'ru',
};

const defaultContext = {
  state: defaultState,
  t: (ru, en) => (defaultState.language === 'en' ? en : ru),
  doLogin: vi.fn(),
  continueAsGuest: vi.fn(),
  socialLogin: vi.fn(),
  toggleHighContrast: vi.fn(),
  toggleLanguage: vi.fn(),
  forgotPassword: vi.fn(),
  openLegal: vi.fn(),
  go: vi.fn(),
};

function renderLogin(overrides = {}) {
  useApp.mockReturnValue({
    ...defaultContext,
    state: { ...defaultState, ...overrides },
  });
  return render(
    <AppProvider>
      <Login />
    </AppProvider>
  );
}

describe('Login', () => {
  it('renders login form', () => {
    renderLogin();
    expect(screen.getByText('FitPulse')).toBeInTheDocument();
    expect(screen.getByText('Войти')).toBeInTheDocument();
  });

  it('renders guest and social buttons', () => {
    renderLogin();
    expect(screen.getByText('Продолжить как гость')).toBeInTheDocument();
    expect(screen.getByText('Google')).toBeInTheDocument();
    expect(screen.getByText('VK')).toBeInTheDocument();
  });

  it('switches language', () => {
    renderLogin();
    const langButton = screen.getByText('EN');
    langButton.click();
    expect(useApp().toggleLanguage).toHaveBeenCalled();
  });

  it('toggles high contrast', () => {
    renderLogin();
    const hcButton = screen.getByText('Режим высокой контрастности');
    hcButton.click();
    expect(useApp().toggleHighContrast).toHaveBeenCalled();
  });
});
