import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useMedia } from './useMedia';

function makeState(overrides = {}) {
  return {
    customExercises: [],
    trainingVideos: [
      { id: 'v1', title: 'Йога' },
      { id: 'v2', title: 'Стретчинг' },
    ],
    videoIndex: 0,
    selectedVideoId: null,
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
  result = renderHook(() => useMedia({ state, update, notify })).result;
}

function setValue(id, value) {
  const el = document.getElementById(id);
  if (el) el.value = value;
}

beforeEach(() => {
  for (const id of ['customExerciseName', 'customExerciseSets']) {
    let el = document.getElementById(id);
    if (!el) {
      el = document.createElement('input');
      el.id = id;
      document.body.appendChild(el);
    }
    el.value = '';
  }
});

describe('useMedia', () => {
  it('returns all media callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'addCustomExercise',
      'changeVideoModal',
      'closeVideoModal',
      'connectVideo',
      'openVideoModal',
      'removeCustomExercise',
    ]);
  });

  it('addCustomExercise notifies when name missing', () => {
    setup();
    setValue('customExerciseSets', '3');
    result.current.addCustomExercise();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Заполните все поля');
  });

  it('addCustomExercise notifies when sets missing', () => {
    setup();
    setValue('customExerciseName', 'Подтягивания');
    result.current.addCustomExercise();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Заполните все поля');
  });

  it('addCustomExercise adds exercise and clears inputs', () => {
    setup({ customExercises: [{ name: 'Бег', sets: '1' }] });
    setValue('customExerciseName', ' Подтягивания ');
    setValue('customExerciseSets', ' 4 ');
    result.current.addCustomExercise();
    expect(update).toHaveBeenCalledWith({
      customExercises: [
        { name: 'Бег', sets: '1' },
        { name: 'Подтягивания', sets: '4' },
      ],
    });
    expect(document.getElementById('customExerciseName').value).toBe('');
    expect(document.getElementById('customExerciseSets').value).toBe('');
  });

  it('addCustomExercise handles undefined customExercises and missing sets input', () => {
    setup({ customExercises: undefined });
    const origGet = document.getElementById.bind(document);
    let setsCalls = 0;
    document.getElementById = (id) => {
      if (id === 'customExerciseSets') {
        setsCalls += 1;
        return setsCalls === 1 ? { value: ' 4 ' } : null;
      }
      return origGet(id);
    };
    try {
      setValue('customExerciseName', ' Подтягивания ');
      result.current.addCustomExercise();
      expect(update).toHaveBeenCalledWith({
        customExercises: [{ name: 'Подтягивания', sets: '4' }],
      });
      expect(document.getElementById('customExerciseName').value).toBe('');
    } finally {
      document.getElementById = origGet;
    }
  });

  it('removeCustomExercise removes item when confirmed', () => {
    setup({ customExercises: [{ name: 'Бег' }, { name: 'Йога' }] });
    window.confirm = vi.fn().mockReturnValue(true);
    result.current.removeCustomExercise(0);
    expect(update).toHaveBeenCalledWith({
      customExercises: [{ name: 'Йога' }],
    });
  });

  it('removeCustomExercise keeps item when cancelled', () => {
    setup({ customExercises: [{ name: 'Бег' }] });
    window.confirm = vi.fn().mockReturnValue(false);
    result.current.removeCustomExercise(0);
    expect(update).not.toHaveBeenCalled();
  });

  it('removeCustomExercise handles undefined customExercises', () => {
    setup({ customExercises: undefined });
    window.confirm = vi.fn().mockReturnValue(true);
    result.current.removeCustomExercise(0);
    expect(update).toHaveBeenCalledWith({ customExercises: [] });
  });

  it('openVideoModal selects video', () => {
    setup();
    result.current.openVideoModal('v2');
    expect(update).toHaveBeenCalledWith({
      videoIndex: 1,
      selectedVideoId: 'v2',
    });
  });

  it('openVideoModal ignores unknown id', () => {
    setup();
    result.current.openVideoModal('v9');
    expect(update).not.toHaveBeenCalled();
  });

  it('closeVideoModal clears selection', () => {
    setup({ selectedVideoId: 'v1' });
    result.current.closeVideoModal();
    expect(update).toHaveBeenCalledWith({ selectedVideoId: null });
  });

  it('changeVideoModal moves forward', () => {
    setup();
    result.current.changeVideoModal(1);
    expect(update).toHaveBeenCalledWith({
      videoIndex: 1,
      selectedVideoId: 'v2',
    });
  });

  it('changeVideoModal wraps to first video', () => {
    setup({ videoIndex: 1 });
    result.current.changeVideoModal(1);
    expect(update).toHaveBeenCalledWith({
      videoIndex: 0,
      selectedVideoId: 'v1',
    });
  });

  it('changeVideoModal wraps backwards', () => {
    setup({ videoIndex: 0 });
    result.current.changeVideoModal(-1);
    expect(update).toHaveBeenCalledWith({
      videoIndex: 1,
      selectedVideoId: 'v2',
    });
  });

  it('connectVideo explains demo playback', () => {
    setup({ selectedVideoId: 'v2' });
    result.current.connectVideo();
    expect(notify).toHaveBeenCalledWith(expect.stringContaining('Стретчинг'));
    expect(notify).toHaveBeenCalledWith(
      expect.stringContaining('демонстрационный видео-урок')
    );
  });

  it('connectVideo does nothing without selection', () => {
    setup();
    result.current.connectVideo();
    expect(notify).not.toHaveBeenCalled();
  });
});
