import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../services/api';
import { useAI } from './useAI';

vi.mock('../services/api', () => ({
  sendAIMessage: vi.fn(),
  quickAI: vi.fn(),
}));

function makeState(overrides = {}) {
  return {
    messages: [],
    ...overrides,
  };
}

let state;
let update;
let result;

function setup(overrides = {}) {
  state = makeState(overrides);
  update = vi.fn();
  result = renderHook(() => useAI({ state, update })).result;
}

function setValue(id, value) {
  const el = document.getElementById(id);
  if (el) el.value = value;
}

beforeEach(() => {
  vi.resetAllMocks();
  let el = document.getElementById('aiInput');
  if (!el) {
    el = document.createElement('input');
    el.id = 'aiInput';
    document.body.appendChild(el);
  }
  el.value = '';
});

describe('useAI', () => {
  it('returns all ai callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'askAI',
      'askAIForDay',
      'quickAI',
      'sendAI',
    ]);
  });

  it('askAI opens ai screen', () => {
    setup();
    result.current.askAI('привет');
    expect(update).toHaveBeenCalledWith({ screen: 'ai' });
  });

  it('askAIForDay opens ai screen with day', () => {
    setup();
    result.current.askAIForDay(12);
    expect(update).toHaveBeenCalledWith({
      screen: 'ai',
      selectedCalendarDay: 12,
    });
  });

  it('sendAI ignores empty input', async () => {
    setup();
    await act(async () => {
      await result.current.sendAI();
    });
    expect(update).not.toHaveBeenCalled();
    expect(api.sendAIMessage).not.toHaveBeenCalled();
  });

  it('sendAI posts message and stores answer', async () => {
    setup();
    setValue('aiInput', '  Как накачать пресс?  ');
    api.sendAIMessage.mockResolvedValue({
      answer: 'Делай планку',
      classification: 'fitness',
      plan: { id: 1 },
      diet: { kcal: 2200 },
      timestamp: '2026-01-01T00:00:00Z',
    });
    await act(async () => {
      await result.current.sendAI();
    });
    expect(api.sendAIMessage).toHaveBeenCalledWith('Как накачать пресс?');
    expect(update).toHaveBeenNthCalledWith(1, {
      messages: [
        expect.objectContaining({ type: 'user', text: 'Как накачать пресс?' }),
      ],
    });
    expect(update).toHaveBeenNthCalledWith(2, {
      messages: [
        expect.objectContaining({ type: 'user', text: 'Как накачать пресс?' }),
        expect.objectContaining({
          type: 'ai',
          text: 'Делай планку',
          classification: 'fitness',
          plan: { id: 1 },
          diet: { kcal: 2200 },
          timestamp: '2026-01-01T00:00:00Z',
        }),
      ],
      aiClassification: 'fitness',
      aiPlan: { id: 1 },
      aiDiet: { kcal: 2200 },
    });
    expect(document.getElementById('aiInput').value).toBe('');
  });

  it('sendAI falls back to message and defaults', async () => {
    setup({ messages: [{ id: '0', type: 'ai', text: 'привет' }] });
    setValue('aiInput', 'вопрос');
    api.sendAIMessage.mockResolvedValue({ message: 'Ответ сервиса' });
    await act(async () => {
      await result.current.sendAI();
    });
    expect(update).toHaveBeenNthCalledWith(2, {
      messages: [
        { id: '0', type: 'ai', text: 'привет' },
        expect.objectContaining({ type: 'user', text: 'вопрос' }),
        expect.objectContaining({
          type: 'ai',
          text: 'Ответ сервиса',
          classification: null,
          plan: null,
          diet: null,
        }),
      ],
      aiClassification: null,
      aiPlan: null,
      aiDiet: null,
    });
  });

  it('sendAI uses default answer text', async () => {
    setup();
    setValue('aiInput', 'вопрос');
    api.sendAIMessage.mockResolvedValue({});
    await act(async () => {
      await result.current.sendAI();
    });
    expect(update).toHaveBeenNthCalledWith(2, {
      messages: [
        expect.objectContaining({ type: 'user' }),
        expect.objectContaining({ type: 'ai', text: 'Ответ получен' }),
      ],
      aiClassification: null,
      aiPlan: null,
      aiDiet: null,
    });
  });

  it('sendAI appends error message on failure', async () => {
    setup();
    setValue('aiInput', 'вопрос');
    api.sendAIMessage.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.sendAI();
    });
    expect(update).toHaveBeenNthCalledWith(2, {
      messages: [
        expect.objectContaining({ type: 'user', text: 'вопрос' }),
        expect.objectContaining({
          type: 'ai',
          text: 'Ошибка подключения к AI сервису',
        }),
      ],
    });
  });

  it('quickAI posts quick question', async () => {
    setup();
    api.quickAI.mockResolvedValue({ answer: 'Готово', classification: 'plan' });
    await act(async () => {
      await result.current.quickAI('Составь план');
    });
    expect(api.quickAI).toHaveBeenCalledWith('Составь план');
    expect(update).toHaveBeenNthCalledWith(1, {
      messages: [
        expect.objectContaining({ type: 'user', text: 'Составь план' }),
      ],
    });
    expect(update).toHaveBeenNthCalledWith(2, {
      messages: [
        expect.objectContaining({ type: 'user', text: 'Составь план' }),
        expect.objectContaining({ type: 'ai', text: 'Готово' }),
      ],
      aiClassification: 'plan',
      aiPlan: null,
      aiDiet: null,
    });
  });

  it('quickAI appends error message on failure', async () => {
    setup();
    api.quickAI.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.quickAI('план');
    });
    expect(update).toHaveBeenNthCalledWith(2, {
      messages: [
        expect.objectContaining({ type: 'user', text: 'план' }),
        expect.objectContaining({
          type: 'ai',
          text: 'Ошибка подключения к AI сервису',
        }),
      ],
    });
  });

  it('quickAI ignores empty response', async () => {
    setup();
    api.quickAI.mockResolvedValue(null);
    await act(async () => {
      await result.current.quickAI('план');
    });
    expect(update).toHaveBeenCalledTimes(1);
  });
});
