import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../services/api';
import { useHealth } from './useHealth';

vi.mock('../services/api', () => ({
  loadBiometrics: vi.fn(),
  addBiometric: vi.fn(),
  loadBodyComposition: vi.fn(),
  saveBodyComposition: vi.fn(),
  loadConditions: vi.fn(),
  createCondition: vi.fn(),
  deleteCondition: vi.fn(),
  loadMenstrualCycles: vi.fn(),
  createMenstrualCycle: vi.fn(),
  updateMenstrualCycle: vi.fn(),
  deleteMenstrualCycle: vi.fn(),
}));

function makeState(overrides = {}) {
  return {
    bodyComposition: [],
    conditions: [],
    menstrualCycles: [],
    weightHistory: [],
    weight: 70,
    waterIntake: 2,
    waterGoal: 8,
    achievements: [],
    points: 0,
    level: 1,
    ...overrides,
  };
}

let state;
let update;
let notify;
let result;

function setup(overrides = {}) {
  state = makeState(overrides);
  update = vi.fn();
  notify = vi.fn();
  const t = (ru) => ru;
  result = renderHook(() => useHealth({ state, update, notify, t })).result;
}

beforeEach(() => {
  vi.resetAllMocks();
});

describe('useHealth', () => {
  it('returns all health callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'addBiometric',
      'addWeight',
      'createCondition',
      'createMenstrualCycle',
      'deleteCondition',
      'deleteMenstrualCycle',
      'drinkWater',
      'loadBiometrics',
      'loadBodyComposition',
      'loadConditions',
      'loadMenstrualCycles',
      'logWeight',
      'saveBodyComposition',
      'updateMenstrualCycle',
    ]);
  });

  it('loadBiometrics stores records', async () => {
    setup();
    api.loadBiometrics.mockResolvedValue({ records: [{ id: 1 }] });
    await act(async () => {
      await result.current.loadBiometrics();
    });
    expect(update).toHaveBeenCalledWith({ biometrics: [{ id: 1 }] });
  });

  it('loadBiometrics ignores payload without records', async () => {
    setup();
    api.loadBiometrics.mockResolvedValue({});
    await act(async () => {
      await result.current.loadBiometrics();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('loadBiometrics swallows errors', async () => {
    setup();
    api.loadBiometrics.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.loadBiometrics();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('addBiometric reloads list and notifies', async () => {
    setup();
    api.addBiometric.mockResolvedValue({});
    api.loadBiometrics.mockResolvedValue({ records: [] });
    await act(async () => {
      await result.current.addBiometric('weight', 74.2);
    });
    expect(api.addBiometric).toHaveBeenCalledWith('weight', 74.2, 'manual');
    expect(update).toHaveBeenCalledWith({ biometrics: [] });
    expect(notify).toHaveBeenCalledWith('Метрика добавлена');
  });

  it('addBiometric passes device type', async () => {
    setup();
    api.addBiometric.mockResolvedValue({});
    await act(async () => {
      await result.current.addBiometric('weight', 74.2, 'scale');
    });
    expect(api.addBiometric).toHaveBeenCalledWith('weight', 74.2, 'scale');
  });

  it('addBiometric notifies on failure', async () => {
    setup();
    api.addBiometric.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.addBiometric('weight', 74.2);
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Ошибка добавления метрики');
  });

  it('loadBodyComposition stores records', async () => {
    setup();
    api.loadBodyComposition.mockResolvedValue({ records: [{ id: 7 }] });
    await act(async () => {
      await result.current.loadBodyComposition();
    });
    expect(update).toHaveBeenCalledWith({ bodyComposition: [{ id: 7 }] });
  });

  it('loadBodyComposition swallows errors', async () => {
    setup();
    api.loadBodyComposition.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.loadBodyComposition();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('saveBodyComposition appends record', async () => {
    setup({ bodyComposition: [{ id: 1 }] });
    api.saveBodyComposition.mockResolvedValue({ record: { id: 2 } });
    let ok;
    await act(async () => {
      ok = await result.current.saveBodyComposition({ weight_kg: 74 });
    });
    expect(api.saveBodyComposition).toHaveBeenCalledWith({ weight_kg: 74 });
    expect(update).toHaveBeenCalledWith({
      bodyComposition: [{ id: 1 }, { id: 2 }],
    });
    expect(notify).toHaveBeenCalledWith('Запись сохранена');
    expect(ok).toBe(true);
  });

  it('saveBodyComposition handles payload without record', async () => {
    setup();
    api.saveBodyComposition.mockResolvedValue({});
    let ok;
    await act(async () => {
      ok = await result.current.saveBodyComposition({});
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Запись сохранена');
    expect(ok).toBe(true);
  });

  it('saveBodyComposition reports failure', async () => {
    setup();
    api.saveBodyComposition.mockRejectedValue(new Error('boom'));
    let ok;
    await act(async () => {
      ok = await result.current.saveBodyComposition({});
    });
    expect(notify).toHaveBeenCalledWith('Ошибка сохранения');
    expect(ok).toBe(false);
  });

  it('loadConditions stores conditions', async () => {
    setup();
    api.loadConditions.mockResolvedValue({ conditions: [{ id: 1 }] });
    await act(async () => {
      await result.current.loadConditions();
    });
    expect(update).toHaveBeenCalledWith({ conditions: [{ id: 1 }] });
  });

  it('loadConditions swallows errors', async () => {
    setup();
    api.loadConditions.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.loadConditions();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('createCondition appends condition', async () => {
    setup({ conditions: [{ id: 1 }] });
    api.createCondition.mockResolvedValue({ condition: { id: 2 } });
    let ok;
    await act(async () => {
      ok = await result.current.createCondition({ title: 'Астма' });
    });
    expect(api.createCondition).toHaveBeenCalledWith({ title: 'Астма' });
    expect(update).toHaveBeenCalledWith({
      conditions: [{ id: 1 }, { id: 2 }],
    });
    expect(notify).toHaveBeenCalledWith('Состояние добавлено');
    expect(ok).toBe(true);
  });

  it('createCondition reports failure', async () => {
    setup();
    api.createCondition.mockRejectedValue(new Error('boom'));
    let ok;
    await act(async () => {
      ok = await result.current.createCondition({});
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Ошибка добавления состояния');
    expect(ok).toBe(false);
  });

  it('deleteCondition removes item', async () => {
    setup({ conditions: [{ id: 1 }, { id: 2 }] });
    api.deleteCondition.mockResolvedValue({});
    await act(async () => {
      await result.current.deleteCondition(1);
    });
    expect(api.deleteCondition).toHaveBeenCalledWith(1);
    expect(update).toHaveBeenCalledWith({ conditions: [{ id: 2 }] });
    expect(notify).toHaveBeenCalledWith('Состояние удалено');
  });

  it('deleteCondition reports failure', async () => {
    setup();
    api.deleteCondition.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.deleteCondition(1);
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Ошибка удаления состояния');
  });

  it('loadMenstrualCycles stores cycles', async () => {
    setup();
    api.loadMenstrualCycles.mockResolvedValue({ cycles: [{ id: 1 }] });
    await act(async () => {
      await result.current.loadMenstrualCycles();
    });
    expect(update).toHaveBeenCalledWith({ menstrualCycles: [{ id: 1 }] });
  });

  it('loadMenstrualCycles ignores invalid payload', async () => {
    setup();
    api.loadMenstrualCycles.mockResolvedValue({});
    await act(async () => {
      await result.current.loadMenstrualCycles();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('loadMenstrualCycles swallows errors', async () => {
    setup();
    api.loadMenstrualCycles.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.loadMenstrualCycles();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('createMenstrualCycle appends cycle', async () => {
    setup({ menstrualCycles: [{ id: 1 }] });
    api.createMenstrualCycle.mockResolvedValue({ cycle: { id: 2 } });
    let ok;
    await act(async () => {
      ok = await result.current.createMenstrualCycle({
        start_date: '2026-01-01',
      });
    });
    expect(api.createMenstrualCycle).toHaveBeenCalledWith({
      start_date: '2026-01-01',
    });
    expect(update).toHaveBeenCalledWith({
      menstrualCycles: [{ id: 1 }, { id: 2 }],
    });
    expect(notify).toHaveBeenCalledWith('Цикл добавлен');
    expect(ok).toBe(true);
  });

  it('createMenstrualCycle reports failure', async () => {
    setup();
    api.createMenstrualCycle.mockRejectedValue(new Error('boom'));
    let ok;
    await act(async () => {
      ok = await result.current.createMenstrualCycle({});
    });
    expect(notify).toHaveBeenCalledWith('Ошибка добавления цикла');
    expect(ok).toBe(false);
  });

  it('updateMenstrualCycle merges cycle', async () => {
    setup({ menstrualCycles: [{ id: 1, notes: 'a' }] });
    api.updateMenstrualCycle.mockResolvedValue({ cycle: { notes: 'b' } });
    let ok;
    await act(async () => {
      ok = await result.current.updateMenstrualCycle(1, { notes: 'b' });
    });
    expect(api.updateMenstrualCycle).toHaveBeenCalledWith(1, { notes: 'b' });
    expect(update).toHaveBeenCalledWith({
      menstrualCycles: [{ id: 1, notes: 'b' }],
    });
    expect(notify).toHaveBeenCalledWith('Цикл обновлён');
    expect(ok).toBe(true);
  });

  it('updateMenstrualCycle reports failure', async () => {
    setup();
    api.updateMenstrualCycle.mockRejectedValue(new Error('boom'));
    let ok;
    await act(async () => {
      ok = await result.current.updateMenstrualCycle(1, {});
    });
    expect(notify).toHaveBeenCalledWith('Ошибка обновления цикла');
    expect(ok).toBe(false);
  });

  it('deleteMenstrualCycle removes cycle', async () => {
    setup({ menstrualCycles: [{ id: 1 }, { id: 2 }] });
    api.deleteMenstrualCycle.mockResolvedValue({});
    await act(async () => {
      await result.current.deleteMenstrualCycle(1);
    });
    expect(api.deleteMenstrualCycle).toHaveBeenCalledWith(1);
    expect(update).toHaveBeenCalledWith({ menstrualCycles: [{ id: 2 }] });
    expect(notify).toHaveBeenCalledWith('Цикл удалён');
  });

  it('deleteMenstrualCycle reports failure', async () => {
    setup();
    api.deleteMenstrualCycle.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.deleteMenstrualCycle(1);
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Ошибка удаления цикла');
  });

  it('logWeight appends to history without sync', async () => {
    setup({ weightHistory: [] });
    await act(async () => {
      await result.current.logWeight(null, null);
    });
    expect(update).toHaveBeenCalledWith({
      weightHistory: [['Сегодня', '70 кг']],
    });
    expect(api.saveBodyComposition).not.toHaveBeenCalled();
  });

  it('logWeight syncs body composition', async () => {
    setup({ weightHistory: [['Вчера', '70 кг']] });
    api.saveBodyComposition.mockResolvedValue({});
    await act(async () => {
      await result.current.logWeight(175, 74.5);
    });
    expect(update).toHaveBeenCalledWith({
      weightHistory: [
        ['Вчера', '70 кг'],
        ['Сегодня', '74.5 кг'],
      ],
    });
    expect(api.saveBodyComposition).toHaveBeenCalledWith({
      height_cm: 175,
      weight_kg: 74.5,
      recorded_at: expect.any(String),
    });
  });

  it('logWeight ignores sync errors', async () => {
    setup();
    api.saveBodyComposition.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.logWeight(175, 74.5);
    });
    expect(update).toHaveBeenCalledWith({
      weightHistory: [['Сегодня', '74.5 кг']],
    });
  });

  it('drinkWater increases intake', () => {
    setup();
    result.current.drinkWater();
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({
        waterIntake: 3,
        lastWaterUpdate: expect.any(Number),
      })
    );
    expect(notify).not.toHaveBeenCalled();
  });

  it('drinkWater notifies when goal already reached', () => {
    setup({ waterIntake: 8, waterGoal: 8 });
    result.current.drinkWater();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith(
      'Ты уже выполнил норму воды на сегодня! 💧'
    );
  });

  it('drinkWater unlocks achievement and points', () => {
    setup({
      waterIntake: 7,
      waterGoal: 8,
      achievements: [{ name: 'Водный баланс', done: false }],
    });
    result.current.drinkWater();
    expect(notify).toHaveBeenCalledWith(
      '🎉 Отлично! Ты выполнил норму воды на сегодня!'
    );
    expect(notify).toHaveBeenCalledWith('⭐ Новое достижение: Водный баланс!');
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ waterIntake: 8, points: 30 })
    );
  });

  it('drinkWater levels up when points threshold reached', () => {
    setup({ waterIntake: 7, waterGoal: 8, points: 970, level: 1 });
    result.current.drinkWater();
    expect(notify).toHaveBeenCalledWith('🎉 Новый уровень! Уровень 2');
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ points: 0, level: 2 })
    );
  });

  it('addWeight appends default entry to history', () => {
    setup({ weightHistory: [] });
    result.current.addWeight();
    expect(update).toHaveBeenCalledWith({
      weightHistory: [['Сегодня', '74.0 кг']],
    });
  });
});
