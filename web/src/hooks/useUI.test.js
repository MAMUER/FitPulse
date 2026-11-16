import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useUI } from './useUI';

function makeState(overrides = {}) {
  return {
    theme: 'dark',
    language: 'ru',
    highContrast: false,
    profile: { twoFactor: false },
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
  result = renderHook(() => useUI({ state, update, notify })).result;
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useUI', () => {
  it('returns all ui callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'addPlace',
      'closeAchievement',
      'closeMeal',
      'closeMetric',
      'editPlace',
      'nextHack',
      'openMetric',
      'selectPlace',
      'showNotification',
      'toggleHighContrast',
      'toggleLanguage',
      'toggleTheme',
      'toggleTwoFactor',
    ]);
  });

  it('toggleTheme switches to light', () => {
    setup();
    result.current.toggleTheme();
    expect(update).toHaveBeenCalledWith({ theme: 'light' });
  });

  it('toggleTheme switches back to dark', () => {
    setup({ theme: 'light' });
    result.current.toggleTheme();
    expect(update).toHaveBeenCalledWith({ theme: 'dark' });
  });

  it('toggleLanguage switches to english', () => {
    setup();
    result.current.toggleLanguage();
    expect(update).toHaveBeenCalledWith({ language: 'en' });
  });

  it('toggleLanguage switches back to russian', () => {
    setup({ language: 'en' });
    result.current.toggleLanguage();
    expect(update).toHaveBeenCalledWith({ language: 'ru' });
  });

  it('toggleHighContrast enables contrast', () => {
    setup();
    result.current.toggleHighContrast();
    expect(update).toHaveBeenCalledWith({ highContrast: true });
  });

  it('toggleHighContrast disables contrast', () => {
    setup({ highContrast: true });
    result.current.toggleHighContrast();
    expect(update).toHaveBeenCalledWith({ highContrast: false });
  });

  it('toggleTwoFactor enables after confirmation', () => {
    setup();
    window.confirm = vi.fn().mockReturnValue(true);
    result.current.toggleTwoFactor();
    expect(window.confirm).toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({
      profile: { twoFactor: true },
    });
  });

  it('toggleTwoFactor keeps state when cancelled', () => {
    setup();
    window.confirm = vi.fn().mockReturnValue(false);
    result.current.toggleTwoFactor();
    expect(update).not.toHaveBeenCalled();
  });

  it('toggleTwoFactor disables without confirmation', () => {
    setup({ profile: { twoFactor: true } });
    window.confirm = vi.fn().mockReturnValue(false);
    result.current.toggleTwoFactor();
    expect(update).toHaveBeenCalledWith({
      profile: { twoFactor: false },
    });
  });

  it('showNotification sends daily reminders', () => {
    setup();
    result.current.showNotification();
    expect(notify).toHaveBeenCalledWith(
      [
        '💧 Не забудь выпить стакан воды!',
        '🍽️ Время обеда — 13:20',
        '🏋️ Тренировка в 18:30',
        '😴 Пора готовиться ко сну',
      ].join('\n')
    );
  });

  it('closeMetric clears metric', () => {
    setup();
    result.current.closeMetric();
    expect(update).toHaveBeenCalledWith({ selectedMetric: null });
  });

  it('openMetric selects the metric details to display', () => {
    setup();
    result.current.openMetric('pulse');
    expect(update).toHaveBeenCalledWith({ selectedMetric: 'pulse' });
  });

  it('closeMeal clears meal', () => {
    setup();
    result.current.closeMeal();
    expect(update).toHaveBeenCalledWith({ selectedMeal: null });
  });

  it('closeAchievement clears achievement', () => {
    setup();
    result.current.closeAchievement();
    expect(update).toHaveBeenCalledWith({ selectedAchievement: null });
  });

  it('selectPlace updates the selected place', () => {
    setup();
    result.current.selectPlace('Бассейн');
    expect(update).toHaveBeenCalledWith({ selectedPlace: 'Бассейн' });
  });

  it('addPlace appends a pinned place', () => {
    setup();
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Парк')
      .mockReturnValueOnce('fa-tree')
      .mockReturnValueOnce('#6fae20')
      .mockReturnValueOnce('40')
      .mockReturnValueOnce('60');
    result.current.addPlace();
    expect(update).toHaveBeenCalledWith({
      pinnedPlaces: [
        expect.objectContaining({
          name: 'Парк',
          icon: 'fa-tree',
          color: '#6fae20',
          x: 40,
          y: 60,
        }),
      ],
    });
  });

  it('addPlace keeps existing pinned places', () => {
    setup({ pinnedPlaces: [{ name: 'Дом', icon: 'fa-house', x: 20, y: 30 }] });
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Зал')
      .mockReturnValueOnce('fa-dumbbell')
      .mockReturnValueOnce('#19d8da')
      .mockReturnValueOnce('70')
      .mockReturnValueOnce('20');
    result.current.addPlace();
    const payload = update.mock.calls[0][0];
    expect(payload.pinnedPlaces).toHaveLength(2);
    expect(payload.pinnedPlaces[0].name).toBe('Дом');
  });

  it('addPlace ignores cancelled prompts', () => {
    setup();
    window.prompt = vi.fn().mockReturnValue(null);
    result.current.addPlace();
    expect(update).not.toHaveBeenCalled();
  });

  it('editPlace updates a pinned place', () => {
    setup({
      pinnedPlaces: [
        { name: 'Дом', icon: 'fa-house', color: '#19d8da', x: 20, y: 30 },
        { name: 'Зал', icon: 'fa-dumbbell', color: '#6fae20', x: 60, y: 40 },
      ],
    });
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Зал 2')
      .mockReturnValueOnce('#ff6375')
      .mockReturnValueOnce('55')
      .mockReturnValueOnce('45');
    result.current.editPlace(1);
    const payload = update.mock.calls[0][0];
    expect(payload.pinnedPlaces[1]).toMatchObject({
      name: 'Зал 2',
      color: '#ff6375',
      x: 55,
      y: 45,
    });
    expect(payload.pinnedPlaces[0].name).toBe('Дом');
  });

  it('editPlace ignores unknown index', () => {
    setup({ pinnedPlaces: [] });
    window.prompt = vi.fn();
    result.current.editPlace(3);
    expect(update).not.toHaveBeenCalled();
    expect(window.prompt).not.toHaveBeenCalled();
  });

  it('addPlace falls back to defaults for empty answers', () => {
    setup();
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Парк') // name
      .mockReturnValueOnce('') // icon
      .mockReturnValueOnce('') // color
      .mockReturnValueOnce('') // x
      .mockReturnValueOnce(''); // y

    result.current.addPlace();

    expect(update.mock.calls[0][0].pinnedPlaces[0]).toMatchObject({
      name: 'Парк',
      icon: 'fa-location-dot',
      color: '#19d8da',
      x: 50,
      y: 50,
    });
  });

  it('editPlace ignores a cancelled name prompt', () => {
    setup({ pinnedPlaces: [{ name: 'Дом', color: '#19d8da', x: 20, y: 30 }] });
    window.prompt = vi.fn().mockReturnValue(null);

    result.current.editPlace(0);

    expect(update).not.toHaveBeenCalled();
  });

  it('editPlace keeps current values for empty answers', () => {
    setup({ pinnedPlaces: [{ name: 'Дом', color: '#19d8da', x: 20, y: 30 }] });
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Дом 2') // name
      .mockReturnValueOnce('') // color -> keeps
      .mockReturnValueOnce('') // x -> keeps
      .mockReturnValueOnce(''); // y -> keeps

    result.current.editPlace(0);

    expect(update.mock.calls[0][0].pinnedPlaces[0]).toMatchObject({
      name: 'Дом 2',
      color: '#19d8da',
      x: 20,
      y: 30,
    });
  });

  it('addPlace ignores a blank name', () => {
    setup();
    window.prompt = vi.fn().mockReturnValueOnce('   ');

    result.current.addPlace();

    expect(update).not.toHaveBeenCalled();
  });

  it('editPlace handles a missing pinned list', () => {
    setup({ pinnedPlaces: undefined });
    window.prompt = vi.fn();

    result.current.editPlace(0);

    expect(update).not.toHaveBeenCalled();
  });

  it('editPlace falls back to the default color when the place has none', () => {
    setup({ pinnedPlaces: [{ name: 'Дом', x: 20, y: 30 }] });
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Дом 2')
      .mockReturnValueOnce('') // color: place.color undefined -> default
      .mockReturnValueOnce('10')
      .mockReturnValueOnce('20');

    result.current.editPlace(0);

    expect(update.mock.calls[0][0].pinnedPlaces[0]).toMatchObject({
      color: '#19d8da',
      x: 10,
      y: 20,
    });
  });

  it('editPlace keeps the current name for a blank answer', () => {
    setup({ pinnedPlaces: [{ name: 'Дом', color: '#19d8da', x: 20, y: 30 }] });
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('') // name -> keeps item.name
      .mockReturnValueOnce('#fff')
      .mockReturnValueOnce('10')
      .mockReturnValueOnce('20');

    result.current.editPlace(0);

    expect(update.mock.calls[0][0].pinnedPlaces[0].name).toBe('Дом');
  });

  it('nextHack handles a missing hack list', () => {
    setup({ hackIndex: 0, hackList: undefined });

    result.current.nextHack();

    expect(update).toHaveBeenCalledWith({ hackIndex: 0 });
  });

  it('nextHack advances and wraps the tip index', () => {
    setup({ hackIndex: 1, hackList: [['a'], ['b'], ['c']] });
    result.current.nextHack();
    expect(update).toHaveBeenCalledWith({ hackIndex: 2 });
  });

  it('nextHack wraps around at the end', () => {
    setup({ hackIndex: 2, hackList: [['a'], ['b'], ['c']] });
    result.current.nextHack();
    expect(update).toHaveBeenCalledWith({ hackIndex: 0 });
  });
});
