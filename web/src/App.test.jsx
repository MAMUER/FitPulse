import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
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

const renderApp = (overrides = {}, initialEntry) => {
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
    getMonthName: () => 'Ноябрь',
    go: () => {},
    update: () => {},
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
    logout: () => {},
    addMeal: () => {},
    removeMeal: () => {},
    toggleDay: () => {},
    logWeight: () => {},
    submitResetEmail: () => {},
    submitResetCode: () => {},
    submitNewPassword: () => {},
    closeLegal: () => {},
    loadProfile: () => {},
    loadBiometrics: () => {},
    loadTrainingPlans: () => {},
    loadMeals: () => {},
    loadCalendarEvents: () => {},
    loadAchievements: () => {},
    loadVideos: () => {},
    loadConditions: () => {},
    loadMenstrualCycles: () => {},
    loadBodyComposition: () => {},
    saveProfile: () => {},
    addEvent: () => {},
    deleteCalendarEvent: () => {},
    createMeal: () => {},
    addBiometric: () => {},
    generatePlan: () => {},
    completeWorkout: () => {},
    loadProgress: () => {},
  });

  return render(
    <MemoryRouter
      initialEntries={[
        initialEntry ?? (overrides.screen === 'home' ? '/home' : '/'),
      ]}
    >
      <AppProvider>
        <App />
      </AppProvider>
    </MemoryRouter>
  );
};

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders login screen by default', () => {
    renderApp();
    // На логине теперь логотип-картинка с alt="FitPulse", а не текст.
    expect(screen.getByAltText('FitPulse')).toBeInTheDocument();
    expect(screen.getByLabelText('Электронная почта')).toBeInTheDocument();
  });

  it('renders home screen when authenticated', () => {
    renderApp({ screen: 'home', user: { email: 'test@example.com' } });
    expect(screen.getByText('Готов к движению')).toBeInTheDocument();
  });

  it('renders the route while auth state is updating', () => {
    renderApp({ screen: 'login' }, '/home');
    expect(screen.getByText('Готов к движению')).toBeInTheDocument();
  });

  it('keeps profile entry in the home header, not the bottom navigation', () => {
    renderApp({ screen: 'home' });
    const navigation = screen.getByRole('navigation');
    expect(within(navigation).queryByText('Профиль')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Профиль' })).toBeInTheDocument();
  });
});
