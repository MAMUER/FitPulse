import { useCallback } from 'react';
import * as api from '../services/api';
import { getDateKey } from '../utils/helpers';
import { t } from '../utils/i18n';

export function useNutrition({
  state,
  update,
  notify,
  getMonthName: getMonthNameFn,
  language,
}) {
  const loadMeals = useCallback(async () => {
    if (state.guest) {
      update({ mealsList: [] });
      return;
    }

    try {
      const payload = await api.loadMeals();
      // Бэкенд отдаёт { status, meals: [...] }; старые версии — просто массив.
      let list = null;
      if (Array.isArray(payload)) {
        list = payload;
      } else if (Array.isArray(payload?.meals)) {
        list = payload.meals;
      }
      if (list) update({ mealsList: list });
    } catch {
      notify('Ошибка загрузки блюд');
    }
  }, [state.guest, update, notify]);

  const createMeal = useCallback(
    async (name, calories, time) => {
      try {
        await api.createMeal(name, calories, time);
        await loadMeals();
        return true;
      } catch {
        notify('Ошибка добавления блюда');
        return false;
      }
    },
    [loadMeals, notify]
  );

  const removeMeal = useCallback(
    async (mealId) => {
      try {
        await api.removeMeal(mealId);
        await loadMeals();
        return true;
      } catch {
        notify('Ошибка удаления блюда');
        return false;
      }
    },
    [loadMeals, notify]
  );

  const updateMeal = useCallback(
    async (mealId, meal) => {
      try {
        await api.updateMeal(mealId, meal);
        await loadMeals();
        notify('Запись обновлена');
        return true;
      } catch {
        notify('Ошибка обновления записи');
        return false;
      }
    },
    [loadMeals, notify]
  );

  const setNutritionPeriod = useCallback(
    (period) => {
      update({ nutritionPeriod: period });
    },
    [update]
  );

  const loadDishes = useCallback(
    async (slot = '', q = '') => {
      if (state.guest) {
        update({ dishes: [] });
        return;
      }
      try {
        const data = await api.loadDishes(slot, q);
        if (data?.dishes) update({ dishes: data.dishes });
      } catch {
        // ignore — dish catalog stays as-is when backend is unavailable
      }
    },
    [state.guest, update]
  );

  // addDishToCatalog добавляет блюдо в каталог пользователя (виден только ему)
  // и обновляет список. Возвращает true при успехе.
  const addDishToCatalog = useCallback(
    async (dish) => {
      if (state.guest) {
        notify('Войдите, чтобы добавлять свои блюда');
        return false;
      }
      try {
        await api.addDish(dish);
        await loadDishes();
        notify('Блюдо добавлено в ваш каталог');
        return true;
      } catch {
        notify('Ошибка добавления блюда');
        return false;
      }
    },
    [state.guest, loadDishes, notify]
  );

  // saveNutritionGoals сохраняет цели по калориям и БЖУ (локально, для сводки).
  const saveNutritionGoals = useCallback(
    (goals) => {
      update({ nutritionGoals: goals });
      notify('Цели питания сохранены');
    },
    [update, notify]
  );

  const setTrainingPeriod = useCallback(
    (period) => {
      update({ trainingPeriod: period });
    },
    [update]
  );

  const openMeal = useCallback(
    (slot) => {
      update({ selectedMeal: slot });
    },
    [update]
  );

  const selectMeal = useCallback(
    (slot, title, kcal, macros, image) => {
      const newMeals = { ...state.meals };
      newMeals[slot] = { ...newMeals[slot], title, kcal, macros, image };
      const d = new Date();
      const key = getDateKey(d.getFullYear(), d.getMonth(), d.getDate());
      const newEvents = { ...state.calendarEvents };
      if (!newEvents[key]) newEvents[key] = [];
      newEvents[key] = [
        ...newEvents[key],
        {
          type: 'food',
          title,
          info: `${kcal} ${t('ккал', 'kcal')} · ${macros}`,
          time: newMeals[slot].time,
          date: `${d.getDate()} ${getMonthNameFn(d.getMonth(), language)}`,
          color: '#f5d45d',
        },
      ];
      update({
        meals: newMeals,
        selectedMeal: null,
        calendarEvents: newEvents,
      });
    },
    [state.meals, state.calendarEvents, language, update, getMonthNameFn]
  );

  const saveCustomMeal = useCallback(() => {
    const name =
      document.getElementById('customName')?.value.trim() || 'Своё блюдо';
    const kcal = Number(document.getElementById('customKcal')?.value) || 0;
    const macros =
      document.getElementById('customMacros')?.value || 'Б 0 г · Ж 0 г · У 0 г';
    const slot = document.getElementById('customSlot')?.value || 'breakfast';
    const newMeals = { ...state.meals };
    newMeals[slot] = { ...newMeals[slot], title: name, kcal, macros };
    const d = new Date();
    const key = getDateKey(d.getFullYear(), d.getMonth(), d.getDate());
    const newEvents = { ...state.calendarEvents };
    if (!newEvents[key]) newEvents[key] = [];
    newEvents[key] = [
      ...newEvents[key],
      {
        type: 'food',
        title: name,
        info: `${kcal} ${t('ккал', 'kcal')} · ${macros}`,
        time: newMeals[slot].time,
        date: `${d.getDate()} ${getMonthNameFn(d.getMonth(), language)}`,
        color: '#f5d45d',
      },
    ];
    update({ meals: newMeals, calendarEvents: newEvents });
    notify('Блюдо добавлено в меню и календарь');
  }, [
    state.meals,
    state.calendarEvents,
    language,
    update,
    notify,
    getMonthNameFn,
  ]);

  return {
    loadMeals,
    loadDishes,
    addDishToCatalog,
    saveNutritionGoals,
    createMeal,
    removeMeal,
    updateMeal,
    setNutritionPeriod,
    setTrainingPeriod,
    openMeal,
    selectMeal,
    saveCustomMeal,
  };
}
