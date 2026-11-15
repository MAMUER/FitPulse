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

  const openMetric = useCallback(
    (metric) => {
      update({ selectedMetric: metric });
    },
    [update]
  );

  const closeMeal = useCallback(() => {
    update({ selectedMeal: null });
  }, [update]);

  const closeAchievement = useCallback(() => {
    update({ selectedAchievement: null });
  }, [update]);

  const selectPlace = useCallback(
    (place) => {
      update({ selectedPlace: place });
    },
    [update]
  );

  const addPlace = useCallback(() => {
    const name = prompt('Название точки', 'Новое место');
    if (!name?.trim()) return;
    const icon = (
      prompt('Иконка Font Awesome', 'fa-location-dot') || 'fa-location-dot'
    ).trim();
    const color =
      prompt('Цвет точки (например #19d8da)', '#19d8da') || '#19d8da';
    const x = Math.max(
      3,
      Math.min(97, Number(prompt('Позиция по горизонтали 0-100', '50')) || 50)
    );
    const y = Math.max(
      3,
      Math.min(90, Number(prompt('Позиция по вертикали 0-100', '50')) || 50)
    );
    update({
      pinnedPlaces: [
        ...(state.pinnedPlaces || []),
        { name: name.trim(), icon, color, x, y },
      ],
    });
  }, [state.pinnedPlaces, update]);

  const editPlace = useCallback(
    (index) => {
      const places = state.pinnedPlaces || [];
      const place = places[index];
      if (!place) return;
      const name = prompt('Название точки', place.name);
      if (name === null) return;
      const color =
        prompt('Цвет точки', place.color || '#19d8da') ||
        place.color ||
        '#19d8da';
      const x = Math.max(
        3,
        Math.min(
          97,
          Number(prompt('Позиция по горизонтали 0-100', String(place.x))) ||
            place.x
        )
      );
      const y = Math.max(
        3,
        Math.min(
          90,
          Number(prompt('Позиция по вертикали 0-100', String(place.y))) ||
            place.y
        )
      );
      update({
        pinnedPlaces: places.map((item, itemIndex) =>
          itemIndex === index
            ? { ...item, name: name.trim() || item.name, color, x, y }
            : item
        ),
      });
    },
    [state.pinnedPlaces, update]
  );

  const nextHack = useCallback(() => {
    const total = state.hackList?.length || 1;
    update({ hackIndex: ((state.hackIndex || 0) + 1) % total });
  }, [state.hackIndex, state.hackList, update]);

  return {
    toggleTheme,
    toggleLanguage,
    toggleHighContrast,
    toggleTwoFactor,
    showNotification,
    openMetric,
    closeMetric,
    closeMeal,
    closeAchievement,
    selectPlace,
    addPlace,
    editPlace,
    nextHack,
  };
}
