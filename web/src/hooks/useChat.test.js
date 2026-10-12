import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useChat } from './useChat';

function makeState(overrides = {}) {
  return {
    guest: false,
    chatMessages: [],
    chatSettings: {
      onlineStatus: true,
      readReceipts: true,
      notifications: true,
    },
    selectedChat: 'Анна',
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
  result = renderHook(() => useChat({ state, update, notify })).result;
}

function setValue(id, value) {
  const el = document.getElementById(id);
  if (el) el.value = value;
}

beforeEach(() => {
  vi.useRealTimers();
  for (const id of [
    'chatInput',
    'csOnline',
    'csRead',
    'csNotify',
    'csConsent',
  ]) {
    let el = document.getElementById(id);
    if (!el) {
      el = document.createElement('input');
      el.id = id;
      document.body.appendChild(el);
    }
    el.value = '';
    el.checked = false;
  }
});

afterEach(() => {
  vi.useRealTimers();
});

describe('useChat', () => {
  it('returns all chat callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'addStickerToChat',
      'openChat',
      'saveChatSettings',
      'sendChat',
    ]);
  });

  it('openChat selects chat and opens detail', () => {
    setup();
    result.current.openChat('Иван');
    expect(update).toHaveBeenCalledWith({
      selectedChat: 'Иван',
      screen: 'chatDetail',
    });
  });

  it('sendChat blocks guest mode', () => {
    setup({ guest: true });
    setValue('chatInput', 'привет');
    result.current.sendChat();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith(
      'В гостевом режиме личный чат недоступен.'
    );
  });

  it('sendChat ignores empty message', () => {
    setup();
    result.current.sendChat();
    expect(update).not.toHaveBeenCalled();
  });

  it('sendChat appends user message and consent', () => {
    setup();
    setValue('chatInput', '  Привет!  ');
    result.current.sendChat();
    expect(update).toHaveBeenCalledWith({
      chatMessages: [
        expect.objectContaining({ from: 'Я', text: 'Привет!', sent: true }),
      ],
      chatConsent: true,
    });
    expect(document.getElementById('chatInput').value).toBe('');
  });

  it('sendChat schedules reply from selected chat', () => {
    vi.useFakeTimers();
    setup();
    setValue('chatInput', 'Привет!');
    act(() => {
      result.current.sendChat();
    });
    expect(update).toHaveBeenCalledTimes(1);
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(update).toHaveBeenCalledTimes(2);
    expect(update).toHaveBeenLastCalledWith({
      chatMessages: [
        expect.objectContaining({
          from: 'Анна',
          text: 'Отлично! Продолжай в том же духе 💪',
          sent: false,
        }),
      ],
    });
  });

  it('sendChat falls back to default sender', () => {
    vi.useFakeTimers();
    setup({ selectedChat: null });
    setValue('chatInput', 'Привет!');
    act(() => {
      result.current.sendChat();
    });
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(update).toHaveBeenLastCalledWith({
      chatMessages: [expect.objectContaining({ from: 'Анна', sent: false })],
    });
  });

  it('saveChatSettings stores checkboxes', () => {
    setup({ chatSettings: { onlineStatus: false } });
    for (const id of ['csOnline', 'csRead', 'csNotify', 'csConsent']) {
      document.getElementById(id).checked = true;
    }
    result.current.saveChatSettings();
    expect(update).toHaveBeenCalledWith({
      chatSettings: {
        onlineStatus: true,
        readReceipts: true,
        notifications: true,
      },
      chatConsent: true,
    });
  });

  it('saveChatSettings stores cleared checkboxes', () => {
    setup();
    result.current.saveChatSettings();
    expect(update).toHaveBeenCalledWith({
      chatSettings: {
        onlineStatus: false,
        readReceipts: false,
        notifications: false,
      },
      chatConsent: false,
    });
  });

  it('addStickerToChat appends sticker to input', () => {
    setup();
    const input = document.getElementById('chatInput');
    input.focus = vi.fn();
    setValue('chatInput', 'Молодец');
    result.current.addStickerToChat(' 💪');
    expect(input.value).toBe('Молодец 💪');
    expect(input.focus).toHaveBeenCalled();
  });

  it('addStickerToChat does nothing without input', () => {
    setup();
    document.getElementById('chatInput').remove();
    expect(() => result.current.addStickerToChat('💪')).not.toThrow();
    const el = document.createElement('input');
    el.id = 'chatInput';
    document.body.appendChild(el);
  });
});
