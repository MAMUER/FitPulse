import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../services/api';
import { getDateKey } from '../utils/helpers';
import { useCalendar } from './useCalendar';

vi.mock('../services/api', () => ({
  createEvent: vi.fn(),
  updateEvent: vi.fn(),
  deleteCalendarEvent: vi.fn(),
}));

function makeState(overrides = {}) {
  return {
    calendarMonthIndex: 9,
    calendarYear: 2026,
    calendarEvents: {},
    calendarEventsList: [],
    selectedCalendarDay: null,
    ...overrides,
  };
}

let state;
let update;
let notify;
let getMonthName;
let result;

function setup(overrides = {}) {
  state = makeState(overrides);
  update = vi.fn();
  notify = vi.fn();
  getMonthName = vi.fn(() => 'Октябрь');
  result = renderHook(() =>
    useCalendar({ state, update, notify, getMonthName, language: 'ru' })
  ).result;
}

function todayKey() {
  const d = new Date();
  return getDateKey(d.getFullYear(), d.getMonth(), d.getDate());
}

function todayDateLabel() {
  const d = new Date();
  return `${d.getDate()} ${getMonthName(d.getMonth(), 'ru')}`;
}

function setValue(id, value) {
  const el = document.getElementById(id);
  if (el) el.value = value;
}

beforeEach(() => {
  vi.resetAllMocks();
  for (const id of [
    'eventType',
    'eventTitle',
    'eventTime',
    'eventDescription',
  ]) {
    let el = document.getElementById(id);
    if (!el) {
      el = document.createElement('input');
      el.id = id;
      document.body.appendChild(el);
    }
    el.value = '';
  }
});

describe('useCalendar', () => {
  it('returns all calendar callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'addEvent',
      'addEventToDay',
      'changeCalendar',
      'closeDateModal',
      'deleteCalendarEvent',
      'openCalendarPage',
      'quickAddEvent',
      'selectCalendarDay',
      'updateEvent',
    ]);
  });

  it('changeCalendar moves to next month', () => {
    setup();
    result.current.changeCalendar(1);
    expect(update).toHaveBeenCalledWith({
      calendarMonthIndex: 10,
      calendarYear: 2026,
      selectedCalendarDay: null,
    });
  });

  it('changeCalendar wraps to previous year', () => {
    setup({ calendarMonthIndex: 0, calendarYear: 2026 });
    result.current.changeCalendar(-1);
    expect(update).toHaveBeenCalledWith({
      calendarMonthIndex: 11,
      calendarYear: 2025,
      selectedCalendarDay: null,
    });
  });

  it('changeCalendar wraps to next year', () => {
    setup({ calendarMonthIndex: 11, calendarYear: 2025 });
    result.current.changeCalendar(1);
    expect(update).toHaveBeenCalledWith({
      calendarMonthIndex: 0,
      calendarYear: 2026,
      selectedCalendarDay: null,
    });
  });

  it('changeCalendar rejects dates before August 2025', () => {
    setup({ calendarMonthIndex: 7, calendarYear: 2025 });
    result.current.changeCalendar(-1);
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith(
      'Нельзя выбрать дату раньше августа 2025 года'
    );
  });

  it('changeCalendar accepts default direction', () => {
    setup();
    result.current.changeCalendar();
    expect(update).toHaveBeenCalledWith({
      calendarMonthIndex: 9,
      calendarYear: 2026,
      selectedCalendarDay: null,
    });
  });

  it('quickAddEvent adds training event for today', () => {
    setup();
    result.current.quickAddEvent('gym');
    expect(update).toHaveBeenCalledWith({
      calendarEvents: {
        [todayKey()]: [
          {
            title: 'Тренировка в зале',
            type: 'gym',
            date: todayDateLabel(),
            time: '19:00',
            color: '#6fae20',
          },
        ],
      },
    });
    expect(getMonthName).toHaveBeenCalledWith(expect.any(Number), 'ru');
  });

  it('quickAddEvent uses fallback label and color', () => {
    setup();
    result.current.quickAddEvent('custom');
    expect(update).toHaveBeenCalledWith({
      calendarEvents: {
        [todayKey()]: [
          expect.objectContaining({
            title: 'custom',
            type: 'custom',
            color: '#19d8da',
          }),
        ],
      },
    });
  });

  it('quickAddEvent appends to existing day', () => {
    setup({
      calendarEvents: {
        [todayKey()]: [
          { title: 'Работа', type: 'work', date: '12 Октябрь', time: '10:00' },
        ],
      },
    });
    result.current.quickAddEvent('pool');
    const payload = update.mock.calls[0][0];
    expect(payload.calendarEvents[todayKey()]).toHaveLength(2);
    expect(payload.calendarEvents[todayKey()][1]).toEqual(
      expect.objectContaining({ type: 'pool', color: '#19d8da' })
    );
  });

  it('openCalendarPage resets to today', () => {
    setup();
    const now = new Date();
    result.current.openCalendarPage();
    expect(update).toHaveBeenCalledWith({
      calendarMonthIndex: now.getMonth(),
      calendarYear: now.getFullYear(),
    });
  });

  it('openCalendarPage keeps month when no reset requested', () => {
    setup();
    result.current.openCalendarPage(false);
    expect(update).not.toHaveBeenCalled();
  });

  it('selectCalendarDay opens day modal', () => {
    setup();
    result.current.selectCalendarDay(15);
    expect(update).toHaveBeenCalledWith({
      selectedCalendarDay: 15,
      dateModalDate: 15,
      showDateModal: true,
    });
  });

  it('closeDateModal closes modal', () => {
    setup();
    result.current.closeDateModal();
    expect(update).toHaveBeenCalledWith({
      showDateModal: false,
      dateModalDate: null,
    });
  });

  it('addEventToDay opens day modal', () => {
    setup();
    result.current.addEventToDay(3);
    expect(update).toHaveBeenCalledWith({
      selectedCalendarDay: 3,
      dateModalDate: 3,
      showDateModal: true,
    });
  });

  it('addEvent creates event from payload', async () => {
    setup({ selectedCalendarDay: 4 });
    api.createEvent.mockResolvedValue({
      event: { id: 42, title: 'План', type: 'training' },
    });
    await act(async () => {
      await result.current.addEvent({
        title: 'План',
        type: 'training',
        time: '18:00',
        description: 'Ноги',
      });
    });
    expect(api.createEvent).toHaveBeenCalledWith({
      title: 'План',
      date: '2026-10-04',
      type: 'training',
      description: 'Ноги',
    });
    expect(update).toHaveBeenCalledWith({
      calendarEvents: {
        '4 Октябрь': [
          expect.objectContaining({
            id: 42,
            title: 'План',
            type: 'training',
            info: 'Ноги',
            time: '18:00',
            date: '4 Октябрь',
            color: '#6fae20',
          }),
        ],
      },
      calendarEventsList: [
        { id: 42, title: 'План', type: 'training', dateLabel: '4 Октябрь' },
      ],
      showDateModal: false,
      dateModalDate: null,
    });
  });

  it('addEvent falls back to DOM inputs and today', async () => {
    setup({ selectedCalendarDay: null });
    const now = new Date();
    setValue('eventType', 'meeting');
    setValue('eventTitle', 'Созвон');
    setValue('eventTime', '09:30');
    setValue('eventDescription', 'С командой');
    api.createEvent.mockResolvedValue({
      event: { id: 7, title: 'Созвон', type: 'meeting' },
    });
    await act(async () => {
      await result.current.addEvent();
    });
    expect(api.createEvent).toHaveBeenCalledWith({
      title: 'Созвон',
      date: `2026-10-${String(now.getDate()).padStart(2, '0')}`,
      type: 'meeting',
      description: 'С командой',
    });
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({
        calendarEventsList: [
          {
            id: 7,
            title: 'Созвон',
            type: 'meeting',
            dateLabel: `${now.getDate()} Октябрь`,
          },
        ],
      })
    );
  });

  it('addEvent uses default title and defaults for missing inputs', async () => {
    setup({ selectedCalendarDay: 1 });
    api.createEvent.mockResolvedValue({ event: { id: 1, title: 't' } });
    await act(async () => {
      await result.current.addEvent({});
    });
    expect(api.createEvent).toHaveBeenCalledWith({
      title: 'Новое событие',
      date: '2026-10-01',
      type: 'training',
      description: '',
    });
  });

  it('addEvent ignores response without event', async () => {
    setup({ selectedCalendarDay: 1 });
    api.createEvent.mockResolvedValue({});
    await act(async () => {
      await result.current.addEvent({ title: 'x' });
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('addEvent notifies on failure', async () => {
    setup({ selectedCalendarDay: 1 });
    api.createEvent.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.addEvent({ title: 'x' });
    });
    expect(notify).toHaveBeenCalledWith('Ошибка добавления события');
  });

  it('updateEvent replaces event in list', async () => {
    setup({
      calendarEventsList: [
        { id: 1, title: 'Старое' },
        { id: 2, title: 'Другое' },
      ],
    });
    api.updateEvent.mockResolvedValue({ event: { id: 1, title: 'Новое' } });
    await act(async () => {
      await result.current.updateEvent(1, 'Новое', '2026-10-01', 'note', 'd');
    });
    expect(api.updateEvent).toHaveBeenCalledWith(
      1,
      'Новое',
      '2026-10-01',
      'note',
      'd'
    );
    expect(update).toHaveBeenCalledWith({
      calendarEventsList: [
        { id: 1, title: 'Новое' },
        { id: 2, title: 'Другое' },
      ],
    });
    expect(notify).toHaveBeenCalledWith('Событие обновлено');
  });

  it('updateEvent ignores response without event', async () => {
    setup();
    api.updateEvent.mockResolvedValue({});
    await act(async () => {
      await result.current.updateEvent(1, 't');
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).not.toHaveBeenCalled();
  });

  it('updateEvent notifies on failure', async () => {
    setup();
    api.updateEvent.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.updateEvent(1, 't');
    });
    expect(notify).toHaveBeenCalledWith('Ошибка обновления события');
  });

  it('deleteCalendarEvent removes event by key and index', async () => {
    setup({
      calendarEvents: { '5 Октябрь': [{ id: 10 }, { id: 11 }] },
      calendarEventsList: [{ id: 10, dateLabel: '5 Октябрь' }],
    });
    api.deleteCalendarEvent.mockResolvedValue({});
    await act(async () => {
      await result.current.deleteCalendarEvent('5 Октябрь', 0);
    });
    expect(api.deleteCalendarEvent).toHaveBeenCalledWith(10);
    expect(update).toHaveBeenCalledWith({
      calendarEvents: { '5 Октябрь': [{ id: 11 }] },
      calendarEventsList: [],
      showDateModal: false,
      dateModalDate: null,
    });
  });

  it('deleteCalendarEvent drops empty day keys', async () => {
    setup({
      calendarEvents: { '5 Октябрь': [{ id: 10 }] },
      calendarEventsList: [],
    });
    api.deleteCalendarEvent.mockResolvedValue({});
    await act(async () => {
      await result.current.deleteCalendarEvent(10);
    });
    expect(api.deleteCalendarEvent).toHaveBeenCalledWith(10);
    expect(update).toHaveBeenCalledWith({
      calendarEvents: {},
      calendarEventsList: [],
      showDateModal: false,
      dateModalDate: null,
    });
  });

  it('deleteCalendarEvent ignores invalid index', async () => {
    setup({
      calendarEvents: { '5 Октябрь': [{ id: 10 }] },
    });
    await act(async () => {
      await result.current.deleteCalendarEvent('5 Октябрь', 5);
    });
    expect(api.deleteCalendarEvent).not.toHaveBeenCalled();
    expect(update).not.toHaveBeenCalled();
  });

  it('deleteCalendarEvent ignores unknown day key', async () => {
    setup({ calendarEvents: {} });
    await act(async () => {
      await result.current.deleteCalendarEvent('нет', 0);
    });
    expect(api.deleteCalendarEvent).not.toHaveBeenCalled();
  });

  it('deleteCalendarEvent notifies on failure', async () => {
    setup();
    api.deleteCalendarEvent.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.deleteCalendarEvent(10);
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Ошибка удаления события');
  });
});
