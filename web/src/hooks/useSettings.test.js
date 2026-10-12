import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../services/api';
import { useSettings } from './useSettings';

vi.mock('../services/api', () => ({
  loadIntegrationProviders: vi.fn(),
  disconnectIntegration: vi.fn(),
}));

function makeState(overrides = {}) {
  return {
    survey: { allergies: [], restrictions: [] },
    progressGoals: {
      weight: {
        selected: 0,
        options: [
          { label: 'Похудеть', days: 30 },
          { label: 'Набрать', days: 60 },
        ],
      },
    },
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
  result = renderHook(() => useSettings({ state, update, notify })).result;
}

beforeEach(() => {
  vi.resetAllMocks();
});

describe('useSettings', () => {
  it('returns all settings callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'chatSettings',
      'disconnectIntegration',
      'loadIntegrationProviders',
      'nutritionSettings',
      'setGoal',
      'showRestrictions',
    ]);
  });

  it('showRestrictions lists survey answers', () => {
    setup({
      survey: { allergies: ['глютен'], restrictions: ['мясо'] },
    });
    result.current.showRestrictions();
    expect(notify).toHaveBeenCalledWith('Аллергии: глютен\nОграничения: мясо');
  });

  it('showRestrictions reports none for empty survey', () => {
    setup();
    result.current.showRestrictions();
    expect(notify).toHaveBeenCalledWith('Аллергии: нет\nОграничения: нет');
  });

  it('chatSettings opens privacy info', () => {
    setup();
    result.current.chatSettings();
    expect(notify).toHaveBeenCalledWith(
      'Настройки чата: приватность, статус, уведомления'
    );
  });

  it('nutritionSettings describes current goal', () => {
    setup({ survey: { allergies: ['глютен'], restrictions: [] } });
    result.current.nutritionSettings();
    expect(notify).toHaveBeenCalledWith(
      expect.stringContaining('Настройки питания')
    );
    expect(notify).toHaveBeenCalledWith(expect.stringContaining('глютен'));
  });

  it('nutritionSettings reports unspecified allergies', () => {
    setup();
    result.current.nutritionSettings();
    expect(notify).toHaveBeenCalledWith(expect.stringContaining('не указаны'));
  });

  it('setGoal selects goal option', () => {
    setup();
    result.current.setGoal('weight', 1);
    expect(notify).toHaveBeenCalledWith('Цель: Набрать (60 дней)');
    expect(update).toHaveBeenCalledWith({
      progressGoals: {
        weight: {
          selected: 1,
          options: [
            { label: 'Похудеть', days: 30 },
            { label: 'Набрать', days: 60 },
          ],
        },
      },
    });
  });

  it('setGoal ignores unknown goal type', () => {
    setup();
    result.current.setGoal('sleep', 0);
    expect(update).not.toHaveBeenCalled();
    expect(notify).not.toHaveBeenCalled();
  });

  it('loadIntegrationProviders returns providers', async () => {
    setup();
    api.loadIntegrationProviders.mockResolvedValue({
      providers: [{ source: 'yandex' }],
    });
    let providers;
    await act(async () => {
      providers = await result.current.loadIntegrationProviders();
    });
    expect(providers).toEqual([{ source: 'yandex' }]);
  });

  it('loadIntegrationProviders returns empty list', async () => {
    setup();
    api.loadIntegrationProviders.mockResolvedValue({});
    let providers;
    await act(async () => {
      providers = await result.current.loadIntegrationProviders();
    });
    expect(providers).toEqual([]);
  });

  it('loadIntegrationProviders swallows errors', async () => {
    setup();
    api.loadIntegrationProviders.mockRejectedValue(new Error('boom'));
    let providers;
    await act(async () => {
      providers = await result.current.loadIntegrationProviders();
    });
    expect(providers).toEqual([]);
  });

  it('disconnectIntegration notifies on success', async () => {
    setup();
    api.disconnectIntegration.mockResolvedValue({});
    await act(async () => {
      await result.current.disconnectIntegration('yandex');
    });
    expect(api.disconnectIntegration).toHaveBeenCalledWith('yandex');
    expect(notify).toHaveBeenCalledWith('Интеграция отключена');
  });

  it('disconnectIntegration notifies on failure', async () => {
    setup();
    api.disconnectIntegration.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.disconnectIntegration('yandex');
    });
    expect(notify).toHaveBeenCalledWith('Не удалось изменить интеграцию');
  });
});
