import { useCallback } from 'react';

export function useUI({ state, update, notify }) {
  const toggleTheme = useCallback(() => {
    update({ theme: state.theme === 'dark' ? 'light' : 'dark' });
  }, [state.theme, update]);

  const toggleLanguage = useCallback(() => {
    update({ language: state.language === 'ru' ? 'en' : 'ru' });
  }, [state.language, update]);

  const toggleHighContrast = useCallback(() => {
    update({ highContrast: !state.highContrast });
  }, [state.highContrast, update]);

  const toggleTwoFactor = useCallback(() => {
    const next = !state.profile.twoFactor;
    if (
      next &&
      !confirm(
        'Включить двухфакторную аутентификацию? Для реальной защиты потребуется серверная проверка второго фактора.'
      )
    )
      return;
    update({ profile: { ...state.profile, twoFactor: next } });
  }, [state.profile, update]);

  const showNotification = useCallback(() => {
    const msgs = [
      '💧 Не забудь выпить стакан воды!',
      '🍽️ Время обеда — 13:20',
      '🏋️ Тренировка в 18:30',
      '😴 Пора готовиться ко сну',
    ];
    notify(msgs.join('\n'));
  }, [notify]);

  const closeMetric = useCallback(() => {
    update({ selectedMetric: null });
  }, [update]);

  const closeMeal = useCallback(() => {
    update({ selectedMeal: null });
  }, [update]);

  const closeAchievement = useCallback(() => {
    update({ selectedAchievement: null });
  }, [update]);

  return {
    toggleTheme,
    toggleLanguage,
    toggleHighContrast,
    toggleTwoFactor,
    showNotification,
    closeMetric,
    closeMeal,
    closeAchievement,
  };
}
