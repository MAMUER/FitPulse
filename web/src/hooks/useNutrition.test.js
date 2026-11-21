import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../services/api';
import { getDateKey } from '../utils/helpers';
import { useNutrition } from './useNutrition';

vi.mock('../services/api', () => ({
  loadMeals: vi.fn(),
  createMeal: vi.fn(),
  removeMeal: vi.fn(),
  updateMeal: vi.fn(),
  loadDishes: vi.fn(),
  addDish: vi.fn(),
}));

function makeState(overrides = {}) {
  return {
    meals: {
      breakfast: { time: '08:00' },
      lunch: { time: '13:00' },
    },
    calendarEvents: {},
    ...overrides,
  };
}

let state;
let update;
let notify;
let getMonthName;
let result;

function setup(overrides = {}) {
  state = makeState(overrides);
  update = vi.fn();
  notify = vi.fn();
  getMonthName = vi.fn(() => 'Октябрь');
  result = renderHook(() =>
    useNutrition({ state, update, notify, getMonthName, language: 'ru' })
  ).result;
}

function todayKey() {
  const d = new Date();
  return getDateKey(d.getFullYear(), d.getMonth(), d.getDate());
}

function todayDateLabel() {
  const d = new Date();
  return `${d.getDate()} ${getMonthName(d.getMonth(), 'ru')}`;
}

function setValue(id, value) {
  const el = document.getElementById(id);
  if (el) el.value = value;
}

beforeEach(() => {
  vi.clearAllMocks();
  globalThis.getDateKey = (year, monthIndex, day) =>
    getDateKey(year, monthIndex, day);
  for (const id of ['customName', 'customKcal', 'customMacros', 'customSlot']) {
    let el = document.getElementById(id);
    if (!el) {
      el = document.createElement('input');
      el.id = id;
      document.body.appendChild(el);
    }
    el.value = '';
  }
});

describe('useNutrition', () => {
  it('returns all nutrition callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'addDishToCatalog',
      'createMeal',
      'loadDishes',
      'loadMeals',
      'openMeal',
      'removeMeal',
      'saveCustomMeal',
      'saveNutritionGoals',
      'selectMeal',
      'setNutritionPeriod',
      'setTrainingPeriod',
      'updateMeal',
    ]);
  });

  it('loads meals into app state', async () => {
    setup();
    const meals = [{ id: 1, name: 'Овсянка' }];
    api.loadMeals.mockResolvedValue(meals);

    await act(async () => result.current.loadMeals());

    expect(update).toHaveBeenCalledWith({ mealsList: meals });
  });

  it('does not request meals for guests', async () => {
    setup({ guest: true });

    await act(async () => result.current.loadMeals());

    expect(api.loadMeals).not.toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({ mealsList: [] });
  });

  it('creates and removes meals, refreshing the list', async () => {
    setup();
    api.createMeal.mockResolvedValue({ id: 1 });
    api.removeMeal.mockResolvedValue({ removed: true });
    api.loadMeals.mockResolvedValue([]);

    await act(async () => result.current.createMeal('Овсянка', 360, '08:00'));
    await act(async () => result.current.removeMeal(1));

    expect(api.createMeal).toHaveBeenCalledWith('Овсянка', 360, '08:00');
    expect(api.removeMeal).toHaveBeenCalledWith(1);
    expect(api.loadMeals).toHaveBeenCalledTimes(2);
  });

  it('setNutritionPeriod updates period', () => {
    setup();
    result.current.setNutritionPeriod('day');
    expect(update).toHaveBeenCalledWith({ nutritionPeriod: 'day' });
  });

  it('setTrainingPeriod updates period', () => {
    setup();
    result.current.setTrainingPeriod('month');
    expect(update).toHaveBeenCalledWith({ trainingPeriod: 'month' });
  });

  it('openMeal selects slot', () => {
    setup();
    result.current.openMeal('lunch');
    expect(update).toHaveBeenCalledWith({ selectedMeal: 'lunch' });
  });

  it('selectMeal fills slot and adds calendar entry', () => {
    setup();
    result.current.selectMeal(
      'lunch',
      'Курица',
      450,
      'Б 30 · Ж 10 · У 5',
      'img'
    );
    expect(update).toHaveBeenCalledWith({
      meals: {
        breakfast: { time: '08:00' },
        lunch: {
          time: '13:00',
          title: 'Курица',
          kcal: 450,
          macros: 'Б 30 · Ж 10 · У 5',
          image: 'img',
        },
      },
      selectedMeal: null,
      calendarEvents: {
        [todayKey()]: [
          {
            type: 'food',
            title: 'Курица',
            info: '450 ккал · Б 30 · Ж 10 · У 5',
            time: '13:00',
            date: todayDateLabel(),
            color: '#f5d45d',
          },
        ],
      },
    });
    expect(getMonthName).toHaveBeenCalledWith(expect.any(Number), 'ru');
  });

  it('selectMeal appends to existing events', () => {
    setup({
      calendarEvents: {
        [todayKey()]: [{ type: 'food', title: 'Завтрак', time: '08:00' }],
      },
    });
    result.current.selectMeal('breakfast', 'Овсянка', 300, 'Б 10', null);
    const payload = update.mock.calls[0][0];
    expect(payload.calendarEvents[todayKey()]).toHaveLength(2);
    expect(payload.calendarEvents[todayKey()][1]).toEqual(
      expect.objectContaining({ title: 'Овсянка', time: '08:00' })
    );
  });

  it('saveCustomMeal stores meal from inputs', () => {
    setup();
    setValue('customName', 'Омлет');
    setValue('customKcal', '320');
    setValue('customMacros', 'Б 20 · Ж 15 · У 3');
    setValue('customSlot', 'breakfast');
    result.current.saveCustomMeal();
    expect(update).toHaveBeenCalledWith({
      meals: {
        breakfast: {
          time: '08:00',
          title: 'Омлет',
          kcal: 320,
          macros: 'Б 20 · Ж 15 · У 3',
        },
        lunch: { time: '13:00' },
      },
      calendarEvents: {
        [todayKey()]: [
          expect.objectContaining({
            title: 'Омлет',
            info: '320 ккал · Б 20 · Ж 15 · У 3',
            time: '08:00',
          }),
        ],
      },
    });
    expect(notify).toHaveBeenCalledWith('Блюдо добавлено в меню и календарь');
  });

  it('saveCustomMeal appends to existing day events', () => {
    setup({
      calendarEvents: {
        [todayKey()]: [{ type: 'food', title: 'Завтрак', time: '08:00' }],
      },
    });
    setValue('customName', 'Омлет');
    setValue('customKcal', '320');
    setValue('customMacros', 'Б 20 · Ж 15 · У 3');
    setValue('customSlot', 'breakfast');
    result.current.saveCustomMeal();
    const payload = update.mock.calls[0][0];
    expect(payload.calendarEvents[todayKey()]).toHaveLength(2);
    expect(payload.calendarEvents[todayKey()][0]).toEqual(
      expect.objectContaining({ title: 'Завтрак' })
    );
    expect(payload.calendarEvents[todayKey()][1]).toEqual(
      expect.objectContaining({ title: 'Омлет' })
    );
  });

  it('saveCustomMeal uses defaults for blank fields', () => {
    setup();
    result.current.saveCustomMeal();
    expect(update).toHaveBeenCalledWith({
      meals: {
        breakfast: {
          time: '08:00',
          title: 'Своё блюдо',
          kcal: 0,
          macros: 'Б 0 г · Ж 0 г · У 0 г',
        },
        lunch: { time: '13:00' },
      },
      calendarEvents: {
        [todayKey()]: [
          expect.objectContaining({
            title: 'Своё блюдо',
            info: '0 ккал · Б 0 г · Ж 0 г · У 0 г',
          }),
        ],
      },
    });
  });

  it('saveCustomMeal falls back to breakfast slot', () => {
    setup();
    setValue('customName', 'Смузи');
    setValue('customKcal', '180');
    setValue('customSlot', '');
    result.current.saveCustomMeal();
    const payload = update.mock.calls[0][0];
    expect(payload.meals.breakfast).toEqual(
      expect.objectContaining({ title: 'Смузи', kcal: 180 })
    );
  });

  it('loadMeals reports a load failure', async () => {
    setup();
    api.loadMeals.mockRejectedValueOnce(new Error('network down'));
    await act(async () => {
      await result.current.loadMeals();
    });
    expect(notify).toHaveBeenCalledWith('Ошибка загрузки блюд');
  });

  it('createMeal returns false and reports a failure', async () => {
    setup();
    api.createMeal.mockRejectedValueOnce(new Error('write failed'));
    let ok;
    await act(async () => {
      ok = await result.current.createMeal('Рис', 300, '12:00');
    });
    expect(ok).toBe(false);
    expect(notify).toHaveBeenCalledWith('Ошибка добавления блюда');
  });

  it('removeMeal returns false and reports a failure', async () => {
    setup();
    api.removeMeal.mockRejectedValueOnce(new Error('delete failed'));
    let ok;
    await act(async () => {
      ok = await result.current.removeMeal(1);
    });
    expect(ok).toBe(false);
    expect(notify).toHaveBeenCalledWith('Ошибка удаления блюда');
  });
});

describe('useNutrition — update, dishes catalog and goals', () => {
  it('loadMeals accepts a bare array payload', async () => {
    setup();
    api.loadMeals.mockResolvedValue([{ id: 1, name: 'Овсянка' }]);
    await act(async () => {
      await result.current.loadMeals();
    });
    expect(update).toHaveBeenCalledWith({
      mealsList: [{ id: 1, name: 'Овсянка' }],
    });
  });

  it('loadMeals accepts the { meals } payload', async () => {
    setup();
    api.loadMeals.mockResolvedValue({ status: 'ok', meals: [{ id: 2 }] });
    await act(async () => {
      await result.current.loadMeals();
    });
    expect(update).toHaveBeenCalledWith({ mealsList: [{ id: 2 }] });
  });

  it('loadMeals ignores payloads without a list', async () => {
    setup();
    api.loadMeals.mockResolvedValue({ status: 'ok' });
    await act(async () => {
      await result.current.loadMeals();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('updateMeal notifies and refreshes on success', async () => {
    setup();
    api.updateMeal.mockResolvedValue({});
    api.loadMeals.mockResolvedValue({ meals: [] });
    let ok = false;
    await act(async () => {
      ok = await result.current.updateMeal('m1', { name: 'Обед' });
    });
    expect(ok).toBe(true);
    expect(api.updateMeal).toHaveBeenCalledWith('m1', { name: 'Обед' });
    expect(notify).toHaveBeenCalledWith('Запись обновлена');
  });

  it('updateMeal reports failures', async () => {
    setup();
    api.updateMeal.mockRejectedValue(new Error('offline'));
    let ok = true;
    await act(async () => {
      ok = await result.current.updateMeal('m1', { name: 'Обед' });
    });
    expect(ok).toBe(false);
    expect(notify).toHaveBeenCalledWith('Ошибка обновления записи');
  });

  it('loadDishes clears the catalog for guests', async () => {
    setup();
    setup({ guest: true });
    await act(async () => {
      await result.current.loadDishes('breakfast', 'овс');
    });
    expect(update).toHaveBeenCalledWith({ dishes: [] });
    expect(api.loadDishes).not.toHaveBeenCalled();
  });

  it('loadDishes stores the catalog', async () => {
    setup();
    api.loadDishes.mockResolvedValue({ dishes: [{ id: 'd1' }] });
    await act(async () => {
      await result.current.loadDishes('breakfast', 'овс');
    });
    expect(api.loadDishes).toHaveBeenCalledWith('breakfast', 'овс');
    expect(update).toHaveBeenCalledWith({ dishes: [{ id: 'd1' }] });
  });

  it('loadDishes ignores errors and payloads without dishes', async () => {
    setup();
    api.loadDishes.mockResolvedValue({});
    await act(async () => {
      await result.current.loadDishes();
    });
    expect(update).not.toHaveBeenCalled();

    api.loadDishes.mockRejectedValue(new Error('offline'));
    await act(async () => {
      await result.current.loadDishes();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('addDishToCatalog refuses for guests', async () => {
    setup();
    setup({ guest: true });
    let ok = true;
    await act(async () => {
      ok = await result.current.addDishToCatalog({ name: 'Борщ' });
    });
    expect(ok).toBe(false);
    expect(notify).toHaveBeenCalledWith('Войдите, чтобы добавлять свои блюда');
    expect(api.addDish).not.toHaveBeenCalled();
  });

  it('addDishToCatalog adds and refreshes', async () => {
    setup();
    api.addDish.mockResolvedValue({});
    api.loadDishes.mockResolvedValue({ dishes: [] });
    let ok = false;
    await act(async () => {
      ok = await result.current.addDishToCatalog({ name: 'Борщ' });
    });
    expect(ok).toBe(true);
    expect(api.addDish).toHaveBeenCalledWith({ name: 'Борщ' });
    expect(notify).toHaveBeenCalledWith('Блюдо добавлено в ваш каталог');
  });

  it('addDishToCatalog reports failures', async () => {
    setup();
    api.addDish.mockRejectedValue(new Error('offline'));
    let ok = true;
    await act(async () => {
      ok = await result.current.addDishToCatalog({ name: 'Борщ' });
    });
    expect(ok).toBe(false);
    expect(notify).toHaveBeenCalledWith('Ошибка добавления блюда');
  });

  it('saveNutritionGoals stores goals and notifies', () => {
    setup();
    act(() => {
      result.current.saveNutritionGoals({ kcal: 2000 });
    });
    expect(update).toHaveBeenCalledWith({ nutritionGoals: { kcal: 2000 } });
    expect(notify).toHaveBeenCalledWith('Цели питания сохранены');
  });
});
