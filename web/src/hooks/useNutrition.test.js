import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getDateKey } from '../utils/helpers';
import { useNutrition } from './useNutrition';

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
      'openMeal',
      'saveCustomMeal',
      'selectMeal',
      'setNutritionPeriod',
      'setTrainingPeriod',
    ]);
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
});
