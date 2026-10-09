import { useCallback } from 'react';
import * as api from '../services/api';
import { t } from '../utils/i18n';

export function useSettings({ state, update, notify }) {
  const showRestrictions = useCallback(() => {
    const a = (state.survey?.allergies || []).join(', ') || t('нет', 'none');
    const r = (state.survey?.restrictions || []).join(', ') || t('нет', 'none');
    notify(
      `${t('Аллергии', 'Allergies')}: ${a}\n${t('Ограничения', 'Restrictions')}: ${r}`
    );
  }, [state.survey, notify]);

  const chatSettings = useCallback(() => {
    notify(
      t(
        'Настройки чата: приватность, статус, уведомления',
        'Chat settings: privacy, status, notifications'
      )
    );
  }, [notify]);

  const nutritionSettings = useCallback(() => {
    notify(
      `${t('Настройки питания', 'Nutrition settings')}: ${t('цель', 'goal')} --- улучшение формы, 2 350 ${t('ккал', 'kcal')}, ${t('белок', 'protein')} 120--145 г, ${t('аллергии', 'allergies')} ${(state.survey?.allergies || []).join(', ') || t('не указаны', 'not specified')}`
    );
  }, [state.survey, notify]);

  const setGoal = useCallback(
    (type, index) => {
      if (state.progressGoals[type]) {
        const opt = state.progressGoals[type].options[index];
        notify(
          `${t('Цель', 'Goal')}: ${opt.label} (${opt.days} ${t('дней', 'days')})`
        );
        const newGoals = { ...state.progressGoals };
        newGoals[type] = { ...newGoals[type], selected: index };
        update({ progressGoals: newGoals });
      }
    },
    [state.progressGoals, notify, update]
  );

  const loadIntegrationProviders = useCallback(async () => {
    try {
      const data = await api.loadIntegrationProviders();
      return data?.providers || [];
    } catch {
      return [];
    }
  }, []);

  const disconnectIntegration = useCallback(
    async (source) => {
      try {
        await api.disconnectIntegration(source);
        notify('Интеграция отключена');
      } catch {
        notify('Не удалось изменить интеграцию');
      }
    },
    [notify]
  );

  return {
    showRestrictions,
    chatSettings,
    nutritionSettings,
    setGoal,
    loadIntegrationProviders,
    disconnectIntegration,
  };
}
