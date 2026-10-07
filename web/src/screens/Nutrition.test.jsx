import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
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
    mealsList: [],
    calorieGoal: 2200,
  },
  t: (ru, _en) => ru,
  createMeal: vi.fn(),
  removeMeal: vi.fn(),
  loadMeals: vi.fn(),
  go: vi.fn(),
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
    <BrowserRouter>
      <AppProvider>
        <Nutrition />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Nutrition', () => {
  it('renders nutrition screen', () => {
    renderNutrition();
    expect(screen.getByText('Питание')).toBeInTheDocument();
  });
});
