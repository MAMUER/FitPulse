import { useCallback } from 'react';
import * as api from '../services/api';

export function useHealth({ state, update, notify, t }) {
  const loadBiometrics = useCallback(async () => {
    try {
      const data = await api.loadBiometrics();
      if (data && Array.isArray(data.records)) {
        update({ biometrics: data.records });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const addBiometric = useCallback(
    async (metricType, value, deviceType = 'manual') => {
      try {
        await api.addBiometric(metricType, value, deviceType);
        await loadBiometrics();
        notify('Метрика добавлена');
      } catch {
        notify('Ошибка добавления метрики');
      }
    },
    [loadBiometrics, notify]
  );

  const loadBodyComposition = useCallback(async () => {
    try {
      const data = await api.loadBodyComposition();
      if (data && Array.isArray(data.records)) {
        update({ bodyComposition: data.records });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const saveBodyComposition = useCallback(
    async (record) => {
      try {
        const data = await api.saveBodyComposition(record);
        if (data?.record) {
          const newRecords = [...(state.bodyComposition || []), data.record];
          update({ bodyComposition: newRecords });
        }
        notify('Запись сохранена');
        return true;
      } catch {
        notify('Ошибка сохранения');
        return false;
      }
    },
    [state.bodyComposition, update, notify]
  );

  const loadConditions = useCallback(async () => {
    try {
      const data = await api.loadConditions();
      if (data && Array.isArray(data.conditions)) {
        update({ conditions: data.conditions });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const createCondition = useCallback(
    async (conditionData) => {
      try {
        const data = await api.createCondition(conditionData);
        if (data?.condition) {
          const newConditions = [...(state.conditions || []), data.condition];
          update({ conditions: newConditions });
        }
        notify('Состояние добавлено');
        return true;
      } catch {
        notify('Ошибка добавления состояния');
        return false;
      }
    },
    [state.conditions, update, notify]
  );

  const deleteCondition = useCallback(
    async (conditionId) => {
      try {
        await api.deleteCondition(conditionId);
        const newConditions = (state.conditions || []).filter(
          (c) => c.id !== conditionId
        );
        update({ conditions: newConditions });
        notify('Состояние удалено');
      } catch {
        notify('Ошибка удаления состояния');
      }
    },
    [state.conditions, update, notify]
  );

  const loadMenstrualCycles = useCallback(async () => {
    try {
      const data = await api.loadMenstrualCycles();
      if (Array.isArray(data?.cycles)) {
        update({ menstrualCycles: data.cycles });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const createMenstrualCycle = useCallback(
    async (cycleData) => {
      try {
        const data = await api.createMenstrualCycle(cycleData);
        if (data?.cycle) {
          const newCycles = [...(state.menstrualCycles || []), data.cycle];
          update({ menstrualCycles: newCycles });
        }
        notify('Цикл добавлен');
        return true;
      } catch {
        notify('Ошибка добавления цикла');
        return false;
      }
    },
    [state.menstrualCycles, update, notify]
  );

  const updateMenstrualCycle = useCallback(
    async (cycleId, cycleData) => {
      try {
        const data = await api.updateMenstrualCycle(cycleId, cycleData);
        if (data?.cycle) {
          const newCycles = (state.menstrualCycles || []).map((c) =>
            c.id === cycleId ? { ...c, ...data.cycle } : c
          );
          update({ menstrualCycles: newCycles });
        }
        notify('Цикл обновлён');
        return true;
      } catch {
        notify('Ошибка обновления цикла');
        return false;
      }
    },
    [state.menstrualCycles, update, notify]
  );

  const deleteMenstrualCycle = useCallback(
    async (cycleId) => {
      try {
        await api.deleteMenstrualCycle(cycleId);
        const newCycles = (state.menstrualCycles || []).filter(
          (c) => c.id !== cycleId
        );
        update({ menstrualCycles: newCycles });
        notify('Цикл удалён');
      } catch {
        notify('Ошибка удаления цикла');
      }
    },
    [state.menstrualCycles, update, notify]
  );

  const logWeight = useCallback(
    async (height, weight) => {
      const newHistory = [
        ...state.weightHistory,
        [t('Сегодня', 'Today'), `${weight || state.weight || 70} кг`],
      ];
      update({ weightHistory: newHistory });
      if (height && weight) {
        try {
          await api.saveBodyComposition({
            height_cm: height,
            weight_kg: weight,
            recorded_at: new Date().toISOString(),
          });
        } catch {
          // ignore
        }
      }
    },
    [state.weightHistory, state.weight, update, t]
  );

  const drinkWater = useCallback(() => {
    if (state.waterIntake < state.waterGoal) {
      const newIntake = state.waterIntake + 1;
      const newState = {
        ...state,
        waterIntake: newIntake,
        lastWaterUpdate: Date.now(),
      };
      if (newIntake === state.waterGoal) {
        notify('🎉 Отлично! Ты выполнил норму воды на сегодня!');
        const ach = newState.achievements.find(
          (a) => a.name === 'Водный баланс'
        );
        if (ach && !ach.done) {
          ach.done = true;
          notify('⭐ Новое достижение: Водный баланс!');
        }
        newState.points += 30;
        if (newState.points >= newState.level * 1000) {
          newState.level++;
          newState.points = 0;
          notify(`🎉 Новый уровень! Уровень ${newState.level}`);
        }
      }
      update(newState);
    } else {
      notify('Ты уже выполнил норму воды на сегодня! 💧');
    }
  }, [state, update, notify]);

  const addWeight = useCallback(() => {
    const newHistory = [
      ...state.weightHistory,
      [t('Сегодня', 'Today'), '74.0 кг'],
    ];
    update({ weightHistory: newHistory });
  }, [state.weightHistory, update, t]);

  return {
    loadBiometrics,
    addBiometric,
    loadBodyComposition,
    saveBodyComposition,
    loadConditions,
    createCondition,
    deleteCondition,
    loadMenstrualCycles,
    createMenstrualCycle,
    updateMenstrualCycle,
    deleteMenstrualCycle,
    logWeight,
    drinkWater,
    addWeight,
  };
}
