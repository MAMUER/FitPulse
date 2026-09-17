import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { AppProvider, useApp } from './contexts/AppContext';

vi.mock('./contexts/AppContext', async () => {
  const actual = await vi.importActual('./contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const renderApp = (overrides = {}) => {
  const defaultState = {
    screen: 'login',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    user: null,
    ...overrides,
  };
  useApp.mockReturnValue({
    state: defaultState,
    t: (ru) => ru,
    go: () => {},
    doLogin: () => {},
    doRegister: () => {},
    continueAsGuest: () => {},
    socialLogin: () => {},
    toggleHighContrast: () => {},
    toggleLanguage: () => {},
    forgotPassword: () => {},
    openLegal: () => {},
    drinkWater: () => {},
    startWorkout: () => {},
    stopWorkout: () => {},
    sendAI: () => {},
    setAI: () => {},
    sendChat: () => {},
    setChat: () => {},
    logout: () => {},
    addMeal: () => {},
    removeMeal: () => {},
    toggleDay: () => {},
    logWeight: () => {},
    submitResetEmail: () => {},
    submitResetCode: () => {},
    submitNewPassword: () => {},
    closeLegal: () => {},
  });

  return render(
    <BrowserRouter>
      <AppProvider>
        <App />
      </AppProvider>
    </BrowserRouter>
  );
};

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders login screen by default', () => {
    renderApp();
    expect(screen.getByText('FitPulse')).toBeInTheDocument();
  });

  it('renders home screen when authenticated', () => {
    renderApp({ screen: 'home', user: { email: 'test@example.com' } });
    expect(screen.getByText('Прогресс')).toBeInTheDocument();
  });

  it('renders tab bar on main screens', () => {
    renderApp({ screen: 'home' });
    const tabs = screen.getAllByRole('button');
    expect(tabs.length).toBeGreaterThan(0);
  });
});
