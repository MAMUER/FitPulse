import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useStories } from './useStories';

const stories = [
  { id: 's1', title: 'Совет дня' },
  { id: 's2', title: 'Тренировка' },
  { id: 's3', title: 'Питание' },
];

function makeState(overrides = {}) {
  return {
    stories,
    storySeen: {},
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
  result = renderHook(() => useStories({ state, update })).result;
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useStories', () => {
  it('returns all story callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'closeStory',
      'nextStory',
      'openStory',
      'prevStory',
      'scrollStories',
      'showAllStories',
    ]);
  });

  it('scrollStories does nothing', () => {
    setup();
    expect(() => result.current.scrollStories('left')).not.toThrow();
    expect(update).not.toHaveBeenCalled();
  });

  it('showAllStories ignores empty list', () => {
    setup({ stories: [] });
    result.current.showAllStories();
    expect(update).not.toHaveBeenCalled();
  });

  it('showAllStories opens first story', () => {
    setup();
    result.current.showAllStories();
    expect(update).toHaveBeenCalledWith({ selectedStory: stories[0] });
  });

  it('openStory selects story and marks it seen', () => {
    setup();
    result.current.openStory(1);
    expect(update).toHaveBeenCalledWith({
      selectedStory: stories[1],
      storySeen: { 1: true },
    });
  });

  it('openStory ignores invalid index', () => {
    setup();
    result.current.openStory(9);
    expect(update).not.toHaveBeenCalled();
  });

  it('openStory merges existing seen flags', () => {
    setup({ storySeen: { 0: true } });
    result.current.openStory(2);
    expect(update).toHaveBeenCalledWith({
      selectedStory: stories[2],
      storySeen: { 0: true, 2: true },
    });
  });

  it('closeStory clears story state', () => {
    setup();
    result.current.closeStory();
    expect(update).toHaveBeenCalledWith({
      selectedStory: null,
      storyPaused: false,
    });
  });

  it('nextStory moves to following story', () => {
    setup({ selectedStory: stories[0], storySeen: { 0: true } });
    result.current.nextStory();
    expect(update).toHaveBeenCalledWith({
      selectedStory: stories[1],
      storySeen: { 0: true, 1: true },
    });
  });

  it('nextStory closes on last story', () => {
    setup({ selectedStory: stories[2] });
    result.current.nextStory();
    expect(update).toHaveBeenCalledWith({
      selectedStory: null,
      storyPaused: false,
    });
  });

  it('nextStory opens first story without selection', () => {
    setup({ selectedStory: null });
    result.current.nextStory();
    expect(update).toHaveBeenCalledWith({
      selectedStory: stories[0],
      storySeen: { 0: true },
    });
  });

  it('prevStory moves to previous story', () => {
    setup({ selectedStory: stories[2], storySeen: {} });
    result.current.prevStory();
    expect(update).toHaveBeenCalledWith({
      selectedStory: stories[1],
      storySeen: { 1: true },
    });
  });

  it('prevStory stops at first story', () => {
    setup({ selectedStory: stories[0] });
    result.current.prevStory();
    expect(update).not.toHaveBeenCalled();
  });

  it('prevStory does nothing without selection', () => {
    setup({ selectedStory: null });
    result.current.prevStory();
    expect(update).not.toHaveBeenCalled();
  });
});
