import { describe, expect, it } from 'vitest';
import { appReducer, initialState } from './appReducer';

const toastState = {
  ...initialState,
  __toast: 'Сохранено',
  __toastTimer: 42,
};

const cases = [
  {
    name: 'UPDATE merges payload',
    action: { type: 'UPDATE', payload: { screen: 'home', xp: 10 } },
    check: (state) => {
      expect(state.screen).toBe('home');
      expect(state.xp).toBe(10);
      expect(state.theme).toBe(initialState.theme);
    },
  },
  {
    name: 'SET_SCREEN changes screen',
    action: { type: 'SET_SCREEN', payload: 'nutrition' },
    check: (state) => {
      expect(state.screen).toBe('nutrition');
      expect(state.theme).toBe(initialState.theme);
    },
  },
  {
    name: 'SET_THEME changes theme',
    action: { type: 'SET_THEME', payload: 'light' },
    check: (state) => expect(state.theme).toBe('light'),
  },
  {
    name: 'SET_LANGUAGE changes language',
    action: { type: 'SET_LANGUAGE', payload: 'en' },
    check: (state) => expect(state.language).toBe('en'),
  },
  {
    name: 'SET_PROFILE merges profile payload',
    action: { type: 'SET_PROFILE', payload: { name: 'Тест', city: 'Москва' } },
    check: (state) => {
      expect(state.profile.name).toBe('Тест');
      expect(state.profile.city).toBe('Москва');
      expect(state.profile.email).toBe(initialState.profile.email);
    },
  },
  {
    name: 'SET_LOADING merges loading payload',
    action: { type: 'SET_LOADING', payload: { profile: true } },
    check: (state) => {
      expect(state.loading).toEqual({ profile: true });
      expect(state.loading.profile).toBe(true);
    },
  },
  {
    name: 'CLEAR_TOAST removes toast state',
    action: { type: 'CLEAR_TOAST' },
    state: toastState,
    check: (state) => {
      expect(state.__toast).toBeNull();
      expect(state.__toastTimer).toBeNull();
      expect(state.screen).toBe(toastState.screen);
    },
  },
];

describe('appReducer', () => {
  it.each(cases)('handles $name', ({ action, state, check }) => {
    const previous = state ?? initialState;
    const next = appReducer(previous, action);
    expect(next).not.toBe(previous);
    check(next);
  });

  it('returns the same state for unknown actions', () => {
    expect(appReducer(initialState, { type: 'UNKNOWN_ACTION' })).toBe(
      initialState
    );
    expect(appReducer(initialState, { type: '' })).toBe(initialState);
  });

  it('does not mutate the previous state', () => {
    const previous = { ...initialState, profile: { ...initialState.profile } };
    appReducer(previous, { type: 'UPDATE', payload: { screen: 'home' } });
    appReducer(previous, { type: 'SET_PROFILE', payload: { name: 'Тест' } });
    appReducer(previous, { type: 'SET_LOADING', payload: { profile: true } });
    expect(previous.screen).toBe(initialState.screen);
    expect(previous.profile.name).toBe(initialState.profile.name);
    expect(previous.loading).toEqual({});
  });
});
