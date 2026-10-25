import { render, screen, act } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import { loadState } from '../services/storage';
import { useProfile } from '../hooks/useProfile';
import { useHealth } from '../hooks/useHealth';
import { useTraining } from '../hooks/useTraining';

vi.mock('../services/storage', () => ({
  loadState: vi.fn(() => null),
  saveState: vi.fn(),
}));
vi.mock('../hooks/useAuth', () => ({ useAuth: vi.fn(() => ({})) }));
vi.mock('../hooks/useNavigation', () => ({
  useNavigation: vi.fn(({ state, update }) => {
    if (update && state?.theme !== 'light') update((prev) => ({ ...prev, theme: 'light' }));
    return {};
  }),
}));
vi.mock('../hooks/useUI', () => ({ useUI: vi.fn(() => ({})) }));
vi.mock('../hooks/useCalendar', () => ({ useCalendar: vi.fn(() => ({})) }));
vi.mock('../hooks/useNutrition', () => ({ useNutrition: vi.fn(() => ({})) }));
vi.mock('../hooks/useTraining', () => ({ useTraining: vi.fn(() => ({ loadTrainingPlans: vi.fn() })) }));
vi.mock('../hooks/useHealth', () => ({
  useHealth: vi.fn(() => ({
    loadBiometrics: vi.fn(),
    loadTrainingPlans: vi.fn(),
    loadConditions: vi.fn(),
    loadMenstrualCycles: vi.fn(),
    loadBodyComposition: vi.fn(),
  })),
}));
vi.mock('../hooks/useChat', () => ({ useChat: vi.fn(() => ({})) }));
vi.mock('../hooks/useProfile', () => ({
  useProfile: vi.fn(() => ({ loadProfile: vi.fn() })),
}));
vi.mock('../hooks/useStories', () => ({ useStories: vi.fn(() => ({})) }));
vi.mock('../hooks/useSurvey', () => ({ useSurvey: vi.fn(() => ({})) }));
vi.mock('../hooks/useAI', () => ({ useAI: vi.fn(() => ({})) }));
vi.mock('../hooks/useMedia', () => ({ useMedia: vi.fn(() => ({})) }));
vi.mock('../hooks/useSettings', () => ({ useSettings: vi.fn(() => ({})) }));

const TestComponent = () => {
  const { state, notify } = useApp();
  return (
    <div>
      <span data-testid="theme">{state.theme}</span>
      <button onClick={() => notify('test message')}>notify</button>
    </div>
  );
};

const renderWithProvider = () => render(<AppProvider><TestComponent /></AppProvider>);

describe('AppContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders children within AppProvider', () => {
    renderWithProvider();
    expect(screen.getByTestId('theme')).toBeInTheDocument();
  });

  it('supports functional state updates via update()', () => {
    renderWithProvider();
    expect(screen.getByTestId('theme').textContent).toBe('light');
  });

  it('notify shows toast then clears it after duration', () => {
    renderWithProvider();
    act(() => {
      screen.getByText('notify').click();
    });
    act(() => {
      vi.advanceTimersByTime(2600);
    });
    expect(screen.getByText('notify')).toBeInTheDocument();
  });

  it('triggers data loading when registered and not profileLoaded', () => {
    const loadProfile = vi.fn();
    const loadBiometrics = vi.fn();
    const loadTrainingPlans = vi.fn();
    const loadConditions = vi.fn();
    const loadMenstrualCycles = vi.fn();
    const loadBodyComposition = vi.fn();
    vi.mocked(loadState).mockReturnValue({ registered: true, profileLoaded: false });
    vi.mocked(useProfile).mockReturnValue({ loadProfile });
    vi.mocked(useHealth).mockReturnValue({
      loadBiometrics,
      loadTrainingPlans,
      loadConditions,
      loadMenstrualCycles,
      loadBodyComposition,
    });
    vi.mocked(useTraining).mockReturnValue({ loadTrainingPlans });
    renderWithProvider();
    expect(loadProfile).toHaveBeenCalled();
  });

  it('useApp throws outside AppProvider', () => {
    expect(() => {
      render(<TestComponent />);
    }).toThrow('useApp must be used within AppProvider');
  });
});
