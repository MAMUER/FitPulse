import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
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
    trainingPlans: [],
  },
  t: (ru, _en) => ru,
  startWorkout: vi.fn(),
  go: vi.fn(),
  loadTrainingPlans: vi.fn(),
  generatePlan: vi.fn(),
  getPlanDetails: vi.fn(),
  completeWorkout: vi.fn(),
};

function renderTraining() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Training />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Training', () => {
  it('renders training screen', () => {
    renderTraining();
    expect(screen.getByText('Тренировки')).toBeInTheDocument();
  });
});
