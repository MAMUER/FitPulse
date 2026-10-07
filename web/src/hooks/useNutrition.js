import { useCallback } from 'react';
import { t } from '../utils/i18n';

export function useNutrition({ state, update, notify, getMonthName: getMonthNameFn, language }) {
  const setNutritionPeriod = useCallback(
    (period) => {
      update({ nutritionPeriod: period });
    },
    [update]
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
  }, [state.meals, state.calendarEvents, language, update, notify, getMonthNameFn]);

  return {
    setNutritionPeriod,
    setTrainingPeriod,
    openMeal,
    selectMeal,
    saveCustomMeal,
  };
}
