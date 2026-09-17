import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Nutrition from './Nutrition';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'nutrition',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    meals: [],
    calorieGoal: 2200,
  },
  t: (ru, _en) => ru,
  addMeal: vi.fn(),
  removeMeal: vi.fn(),
  setNutritionPeriod: vi.fn(),
  drinkWater: vi.fn(),
  nutritionSettings: vi.fn(),
  openMeal: vi.fn(),
  selectMeal: vi.fn(),
  saveCustomMeal: vi.fn(),
  askAI: vi.fn(),
};

function renderNutrition() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <AppProvider>
      <Nutrition />
    </AppProvider>
  );
}

describe('Nutrition', () => {
  it('renders nutrition screen', () => {
    renderNutrition();
    expect(screen.getByText('Питание')).toBeInTheDocument();
  });
});
