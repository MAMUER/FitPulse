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
      'closeAchievement',
      'closeMeal',
      'closeMetric',
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
});
