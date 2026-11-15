import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Survey from './Survey';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

function setup(overrides = {}) {
  const context = {
    state: {
      language: 'ru',
      survey: {
        step: 0,
        allergies: [],
        restrictions: [],
        goals: [],
        diet: '',
        activity: '',
        sleepHours: '',
        waterIntake: '',
        preferredWorkouts: [],
        trainingDays: '',
      },
    },
    t: (ru) => ru,
    go: vi.fn(),
    surveySelect: vi.fn(),
    surveyNext: vi.fn(),
    surveyPrev: vi.fn(),
    deferSurvey: vi.fn(),
    ...overrides,
  };
  useApp.mockReturnValue(context);
  render(<Survey />);
  return context;
}

describe('Survey', () => {
  it('renders the current question and progress', () => {
    setup();

    expect(screen.getByText('Персонализированный опрос')).toBeInTheDocument();
    expect(screen.getByText('1 / 9')).toBeInTheDocument();
    expect(screen.getByLabelText('Орехи')).toBeInTheDocument();
  });

  it('records a selected option and advances', () => {
    const context = setup();

    fireEvent.click(screen.getByLabelText('Орехи'));
    fireEvent.click(screen.getByRole('button', { name: 'Далее' }));

    expect(context.surveySelect).toHaveBeenCalledWith(
      'allergies',
      'Орехи',
      true
    );
    expect(context.surveyNext).toHaveBeenCalledTimes(1);
  });

  it('defers the survey and returns to home', () => {
    const context = setup();

    fireEvent.click(screen.getByRole('button', { name: 'Заполнить позже' }));

    expect(context.deferSurvey).toHaveBeenCalledTimes(1);
    expect(context.go).toHaveBeenCalledWith('home');
  });

  it('returns to home after completing the final question', async () => {
    const context = setup({
      state: {
        language: 'ru',
        survey: { step: 8, trainingDays: '3 дня' },
      },
    });

    fireEvent.click(screen.getByRole('button', { name: 'Завершить' }));

    expect(context.surveyNext).toHaveBeenCalledTimes(1);
    await waitFor(() => {
      expect(context.go).toHaveBeenCalledWith('home');
    });
  });
});
