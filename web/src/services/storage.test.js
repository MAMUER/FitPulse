import { beforeEach, describe, expect, it, vi } from 'vitest';
import { loadState, saveState } from './storage';

const KEY = 'fitpulse-merged-v9';

beforeEach(() => {
  localStorage.clear();
});

describe('loadState', () => {
  it('returns null when nothing is stored', () => {
    expect(loadState()).toBeNull();
  });

  it('parses stored json', () => {
    localStorage.setItem(KEY, JSON.stringify({ screen: 'home', xp: 10 }));
    expect(loadState()).toEqual({ screen: 'home', xp: 10 });
  });

  it('returns null for invalid json', () => {
    localStorage.setItem(KEY, '{invalid-json');
    expect(loadState()).toBeNull();
  });
});

describe('saveState', () => {
  it('serializes state into storage', () => {
    saveState({ screen: 'home' });
    expect(localStorage.getItem(KEY)).toBe('{"screen":"home"}');
  });

  it('serializes nested state', () => {
    saveState({ profile: { name: 'Тест' }, mealsList: [] });
    expect(localStorage.getItem(KEY)).toBe(
      '{"profile":{"name":"Тест"},"mealsList":[]}'
    );
  });

  it('swallows storage errors', () => {
    const setItem = vi.spyOn(localStorage, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError');
    });
    expect(() => saveState({ screen: 'home' })).not.toThrow();
    setItem.mockRestore();
  });
});
