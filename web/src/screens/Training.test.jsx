import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Training from './Training';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'training',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    workoutStartDate: null,
    workoutDuration: 0,
  },
  t: (ru, _en) => ru,
  stopWorkout: vi.fn(),
  startWorkout: vi.fn(),
  go: vi.fn(),
  toggleWorkout: vi.fn(),
  editPlace: vi.fn(),
  selectPlace: vi.fn(),
  addPlace: vi.fn(),
};

function renderTraining() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <AppProvider>
      <Training />
    </AppProvider>
  );
}

describe('Training', () => {
  it('renders training screen', () => {
    renderTraining();
    expect(screen.getByText('Тренировка')).toBeInTheDocument();
  });
});
