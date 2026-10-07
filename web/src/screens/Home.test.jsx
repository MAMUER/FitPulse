import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Home from './Home';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'home',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    weight: 74.2,
    height: 175,
    waterIntake: 2,
    waterGoal: 8,
    profile: { name: 'Михаил', email: 'mih@example.com' },
    biometrics: [],
    trainingPlans: [],
    achievementsBackend: [],
    registered: true,
    profileLoaded: true,
  },
  t: (ru, _en) => ru,
  drinkWater: vi.fn(),
  startWorkout: vi.fn(),
  go: vi.fn(),
  openLegal: vi.fn(),
  loadBiometrics: vi.fn(),
  loadTrainingPlans: vi.fn(),
  loadAchievements: vi.fn(),
  loadProgress: vi.fn(),
};

function renderHome() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Home />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Home', () => {
  it('renders progress and actions', () => {
    renderHome();
    expect(screen.getByText('Прогресс')).toBeInTheDocument();
  });

  it('calls drinkWater', () => {
    renderHome();
    screen.getByText('+250 мл').click();
    expect(defaultContext.drinkWater).toHaveBeenCalled();
  });

  it('navigates to body profile', () => {
    renderHome();
    screen.getByText('Профиль тела').click();
    expect(defaultContext.go).toHaveBeenCalledWith('body');
  });
});
