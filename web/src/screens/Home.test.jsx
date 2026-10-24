import { render, screen } from '@testing-library/react';
import { BrowserRouter, useNavigate } from 'react-router-dom';
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

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: vi.fn(() => vi.fn()),
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

function renderHome(overrides) {
  useApp.mockReturnValue(
    overrides ? { ...defaultContext, ...overrides } : defaultContext
  );
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

  it('renders biometric metrics when present', () => {
    renderHome({
      state: {
        ...defaultContext.state,
        biometrics: [
          { metric_type: 'heart_rate', value: 72 },
          { metric_type: 'spo2', value: 98 },
          { metric_type: 'temperature', value: 36.6 },
        ],
      },
    });
    expect(screen.getByText('Пульс')).toBeInTheDocument();
    expect(screen.getByText('SpO2')).toBeInTheDocument();
    expect(screen.getByText('Температура')).toBeInTheDocument();
    expect(screen.getByText('72')).toBeInTheDocument();
    expect(screen.getByText('98%')).toBeInTheDocument();
    expect(screen.getByText('36.6°')).toBeInTheDocument();
  });

  it('navigates to training plans from workouts panel', () => {
    renderHome();
    screen.getByText('Планы тренировок').click();
    expect(defaultContext.go).toHaveBeenCalledWith('training');
  });

  it('navigates to videos from recommendations panel', () => {
    renderHome();
    screen.getByText('Видео-тренировки').click();
    expect(defaultContext.go).toHaveBeenCalledWith('videos');
  });

  it('navigates to AI advisor from recommendations panel', () => {
    renderHome();
    screen.getByText('AI-советник').click();
    expect(defaultContext.go).toHaveBeenCalledWith('ai');
  });

  it('navigates to consent from policy link in goals panel', () => {
    renderHome();
    screen.getByText('Политика').click();
    const navigate =
      useNavigate.mock.results[useNavigate.mock.results.length - 1].value;
    expect(navigate).toHaveBeenCalledWith('/consent');
  });
});
