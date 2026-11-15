import { act, renderHook } from '@testing-library/react';
import { createElement } from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useNavigation } from './useNavigation';

function Wrapper({ children }) {
  return createElement(MemoryRouter, { initialEntries: ['/home'] }, children);
}

function makeState(overrides = {}) {
  return {
    screen: 'home',
    guest: false,
    legalReturnScreen: null,
    selectedChat: null,
    selectedMeal: null,
    selectedMetric: null,
    showDateModal: false,
    selectedAchievement: null,
    selectedStory: null,
    ...overrides,
  };
}

let state;
let update;
let result;

function setup(overrides = {}) {
  state = makeState(overrides);
  update = vi.fn();
  result = renderHook(
    () => ({
      ...useNavigation({ state, update }),
      location: useLocation(),
    }),
    { wrapper: Wrapper }
  ).result;
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useNavigation', () => {
  it('returns all navigation callbacks', () => {
    setup();
    expect(
      Object.keys(result.current)
        .filter((key) => key !== 'location')
        .sort()
    ).toEqual(['backFromLegal', 'go', 'openLegal']);
  });

  it('go switches screen and closes overlays', () => {
    setup({
      selectedMetric: 'weight',
      showDateModal: true,
      selectedChat: 'Анна',
    });
    act(() => result.current.go('body'));
    expect(update).toHaveBeenCalledWith({
      screen: 'body',
      selectedMeal: null,
      selectedMetric: null,
      showDateModal: false,
      selectedAchievement: null,
      selectedStory: null,
      selectedChat: 'Анна',
    });
  });

  it('go keeps selected chat when leaving another screen', () => {
    setup({ screen: 'chatDetail', selectedChat: 'Анна' });
    act(() => result.current.go('home'));
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ screen: 'home', selectedChat: 'Анна' })
    );
  });

  it('go clears chat selection when entering chat screen', () => {
    setup({ screen: 'home', selectedChat: 'Анна' });
    act(() => result.current.go('chat'));
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ screen: 'chat', selectedChat: null })
    );
  });

  it('go updates the route to match the selected screen', () => {
    setup();
    act(() => result.current.go('nutrition'));
    expect(result.current.location.pathname).toBe('/nutrition');
  });

  it('go ignores same screen without overlays', () => {
    setup({ screen: 'home' });
    act(() => result.current.go('home'));
    expect(update).not.toHaveBeenCalled();
  });

  it('go reopens screen to close meal modal', () => {
    setup({ screen: 'nutrition', selectedMeal: 'lunch' });
    act(() => result.current.go('nutrition'));
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ screen: 'nutrition', selectedMeal: null })
    );
  });

  it('go reopens screen to close story modal', () => {
    setup({ screen: 'home', selectedStory: { id: 's1' } });
    act(() => result.current.go('home'));
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ selectedStory: null })
    );
  });

  it('go reopens screen to close achievement modal', () => {
    setup({ screen: 'profile', selectedAchievement: { name: 'Цель' } });
    act(() => result.current.go('profile'));
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ selectedAchievement: null })
    );
  });

  it('openLegal remembers login as return for privacy', () => {
    setup({ screen: 'ai', guest: false });
    act(() => result.current.openLegal('privacy'));
    expect(update).toHaveBeenCalledWith({
      legalReturnScreen: 'login',
      screen: 'privacy',
    });
    expect(result.current.location.pathname).toBe('/privacy');
  });

  it('openLegal remembers home as return for guest terms', () => {
    setup({ screen: 'login', guest: true });
    act(() => result.current.openLegal('terms'));
    expect(update).toHaveBeenCalledWith({
      legalReturnScreen: 'home',
      screen: 'terms',
    });
  });

  it('openLegal keeps existing return screen', () => {
    setup({ guest: true, legalReturnScreen: 'body' });
    act(() => result.current.openLegal('privacy'));
    expect(update).toHaveBeenCalledWith({
      legalReturnScreen: 'body',
      screen: 'privacy',
    });
  });

  it('openLegal stores current screen for other screens', () => {
    setup({ screen: 'profile', guest: false });
    act(() => result.current.openLegal('support'));
    expect(update).toHaveBeenCalledWith({
      legalReturnScreen: 'profile',
      screen: 'support',
    });
  });

  it('backFromLegal returns to stored screen', () => {
    setup({ legalReturnScreen: 'body', guest: true });
    act(() => result.current.backFromLegal());
    expect(update).toHaveBeenCalledWith({
      legalReturnScreen: null,
      screen: 'body',
    });
    expect(result.current.location.pathname).toBe('/body');
  });

  it('backFromLegal falls back to home for guest', () => {
    setup({ legalReturnScreen: null, guest: true });
    act(() => result.current.backFromLegal());
    expect(update).toHaveBeenCalledWith({
      legalReturnScreen: null,
      screen: 'home',
    });
  });

  it('backFromLegal falls back to login for user', () => {
    setup({ legalReturnScreen: null, guest: false });
    act(() => result.current.backFromLegal());
    expect(update).toHaveBeenCalledWith({
      legalReturnScreen: null,
      screen: 'login',
    });
  });
});
