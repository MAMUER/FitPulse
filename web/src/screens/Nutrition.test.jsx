import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Nutrition from './Nutrition';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

const meals = {
  breakfast: {
    label: 'Завтрак',
    icon: 'fa-mug-hot',
    time: '08:00',
    title: 'Овсянка',
    kcal: 420,
    macros: 'Б 18 г · Ж 12 г · У 60 г',
  },
  lunch: {
    label: 'Обед',
    icon: 'fa-bowl-food',
    time: '13:20',
    title: 'Курица с рисом',
    kcal: 650,
    macros: 'Б 42 г · Ж 18 г · У 72 г',
  },
  snack: {
    label: 'Полдник',
    icon: 'fa-apple-whole',
    time: '16:30',
    title: 'Яблоко',
    kcal: 95,
    macros: 'Б 1 г · Ж 0 г · У 24 г',
  },
  dinner: {
    label: 'Ужин',
    icon: 'fa-fish',
    time: '19:30',
    title: 'Лосось',
    kcal: 520,
    macros: 'Б 38 г · Ж 24 г · У 30 г',
  },
};

const alternatives = {
  breakfast: [
    ['Омлет', 350, 'Б 24 г · Ж 22 г · У 6 г', '/omelet.jpg'],
    ['Сырники', 410, 'Б 26 г · Ж 14 г · У 40 г', '/syrniki.jpg'],
  ],
  lunch: [['Индейка', 590, 'Б 48 г · Ж 12 г · У 55 г', '/turkey.jpg']],
  dinner: [['Треска', 430, 'Б 34 г · Ж 9 г · У 26 г', '/cod.jpg']],
};

function makeContext(overrides = {}) {
  const { state: stateOverrides, t: tOverride, ...rest } = overrides;
  return {
    state: {
      language: 'ru',
      meals,
      alternatives,
      nutritionPeriod: 'day',
      nutritionData: {
        day: {
          calories: 1685,
          goal: 2350,
          protein: 124,
          fats: 66,
          carbs: 188,
        },
        week: {
          calories: 11800,
          goal: 16450,
          protein: 860,
          fats: 470,
          carbs: 1320,
        },
        month: {
          calories: 50200,
          goal: 70500,
          protein: 3700,
          fats: 2010,
          carbs: 5650,
        },
      },
      mealsList: [{ id: 1, name: 'Яйца', calories: 200, time: '08:00' }],
      selectedMeal: null,
      waterIntake: 2,
      waterGoal: 8,
      ...stateOverrides,
    },
    t: tOverride || ((ru, _en) => ru),
    go: vi.fn(),
    loadMeals: vi.fn(),
    openMeal: vi.fn(),
    closeMeal: vi.fn(),
    selectMeal: vi.fn(),
    saveCustomMeal: vi.fn(),
    setNutritionPeriod: vi.fn(),
    nutritionSettings: vi.fn(),
    quickAI: vi.fn(),
    createMeal: vi.fn().mockResolvedValue({}),
    removeMeal: vi.fn().mockResolvedValue({}),
    drinkWater: vi.fn(),
    ...rest,
  };
}

describe('Nutrition', () => {
  it('renders the nutrition plan and loads meals', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);

    expect(context.loadMeals).toHaveBeenCalled();
    expect(screen.getByText('План питания')).toBeInTheDocument();
    expect(screen.getByText('1 685')).toBeInTheDocument();
    expect(screen.getByText(/Овсянка/)).toBeInTheDocument();
    expect(screen.getByText(/Курица с рисом/)).toBeInTheDocument();
    expect(screen.getByText(/Лосось/)).toBeInTheDocument();
    expect(screen.getByText('Яйца')).toBeInTheDocument();
  });

  it('switches the nutrition period', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    fireEvent.click(screen.getByRole('button', { name: 'Неделя' }));

    expect(context.setNutritionPeriod).toHaveBeenCalledWith('week');
  });

  it('opens the meal picker for a slot', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    fireEvent.click(screen.getAllByText('Сменить')[0]);

    expect(context.openMeal).toHaveBeenCalledWith('breakfast');
  });

  it('selects an AI suggestion', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    fireEvent.click(screen.getAllByText('Выбрать')[0]);

    expect(context.selectMeal).toHaveBeenCalledWith(
      'breakfast',
      'Омлет',
      350,
      'Б 24 г · Ж 22 г · У 6 г',
      '/omelet.jpg'
    );
  });

  it('adds a meal from the diary form', async () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    // "Название"/"Калории" exist twice: once in the custom-meal card and once
    // in the diary form. Scope the queries to the diary card.
    const diary = screen.getByText('Добавить в дневник').closest('section');
    fireEvent.change(within(diary).getByLabelText('Название'), {
      target: { value: 'Рис' },
    });
    fireEvent.change(within(diary).getByLabelText('Калории'), {
      target: { value: '300' },
    });
    fireEvent.change(within(diary).getByLabelText('Время'), {
      target: { value: '12:00' },
    });
    fireEvent.click(screen.getByText('Добавить в дневник'));

    // handleAddToDiary awaits createMeal() and then resets the form fields, so
    // the state updates land after the click — wait for them inside act().
    await waitFor(() =>
      expect(context.createMeal).toHaveBeenCalledWith('Рис', 300, '12:00')
    );
  });

  it('adds a meal with defaults when the form is empty', () => {
    const context = makeContext({ state: { mealsList: [] } });
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    fireEvent.click(screen.getByText('Добавить в дневник'));

    expect(context.createMeal).toHaveBeenCalledWith(
      'Блюдо',
      0,
      expect.any(String)
    );
  });

  it('removes a meal from the diary', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    fireEvent.click(screen.getByLabelText('Удалить'));

    expect(context.removeMeal).toHaveBeenCalledWith(1);
  });

  it('asks AI for a menu', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    fireEvent.click(screen.getByText('Спросить AI'));

    expect(context.go).toHaveBeenCalledWith('ai');
    expect(context.quickAI).toHaveBeenCalledWith('Составь меню на сегодня');
  });

  it('shows the dish modal for the selected slot', () => {
    const context = makeContext({ state: { selectedMeal: 'lunch' } });
    useApp.mockReturnValue(context);

    render(<Nutrition />);

    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByText('Выбери блюдо')).toBeInTheDocument();
    expect(within(dialog).getByText('Курица с рисом')).toBeInTheDocument();
    expect(within(dialog).getByText('Индейка')).toBeInTheDocument();
    fireEvent.click(within(dialog).getByLabelText('Закрыть'));

    expect(context.closeMeal).toHaveBeenCalled();
  });

  it('selects a dish in the modal', () => {
    const context = makeContext({ state: { selectedMeal: 'lunch' } });
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    const dialog = screen.getByRole('dialog');
    fireEvent.click(within(dialog).getByText('Выбрать'));

    expect(context.selectMeal).toHaveBeenCalledWith(
      'lunch',
      'Индейка',
      590,
      'Б 48 г · Ж 12 г · У 55 г',
      '/turkey.jpg'
    );
  });

  it('opens personal nutrition settings', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    fireEvent.click(screen.getByText('Персональные настройки'));

    expect(context.nutritionSettings).toHaveBeenCalled();
  });

  it('saves a custom meal', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Nutrition />);
    fireEvent.click(screen.getByText('Сохранить в меню и календарь'));

    expect(context.saveCustomMeal).toHaveBeenCalled();
  });

  it('renders an empty diary when there are no entries', () => {
    const context = makeContext({ state: { mealsList: null } });
    useApp.mockReturnValue(context);

    render(<Nutrition />);

    expect(screen.getByText('Записей пока нет')).toBeInTheDocument();
  });
});
