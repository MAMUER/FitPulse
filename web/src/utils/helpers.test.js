import { describe, expect, it } from 'vitest';
import * as helpers from './helpers';
import { getMonthName } from './i18n';

const { MIN_MONTH, MIN_YEAR } = helpers;

describe('constants', () => {
  it('exposes the allowed range boundaries', () => {
    expect(MIN_YEAR).toBe(2025);
    expect(MIN_MONTH).toBe(7);
  });
});

describe('week and date keys', () => {
  it('returns week days for a language', () => {
    expect(helpers.getWeekDays('en')).toEqual([
      'Mon',
      'Tue',
      'Wed',
      'Thu',
      'Fri',
      'Sat',
      'Sun',
    ]);
    expect(helpers.getWeekDays('ru')).toEqual([
      'Пн',
      'Вт',
      'Ср',
      'Чт',
      'Пт',
      'Сб',
      'Вс',
    ]);
    expect(helpers.getWeekDays()).toEqual(helpers.getWeekDays('ru'));
    expect(helpers.getWeekDays('fr')).toEqual(helpers.getWeekDays('ru'));
  });

  it('builds padded date keys', () => {
    expect(helpers.getDateKey(2025, 6, 5)).toBe('2025-07-05');
    expect(helpers.getDateKey(2025, 11, 31)).toBe('2025-12-31');
    expect(helpers.getDateKey(2025, 0, 1)).toBe('2025-01-01');
  });
});

describe('calendar events', () => {
  const calendarEvents = {
    '2025-07-05': [{ title: 'Тренировка' }],
    '2025-07-24': [{ title: 'Питание' }, { title: 'Восстановление' }],
  };

  it('returns events for a single date', () => {
    expect(helpers.getEventsForDate(calendarEvents, 2025, 6, 5)).toEqual([
      { title: 'Тренировка' },
    ]);
    expect(helpers.getEventsForDate(calendarEvents, 2025, 6, 6)).toEqual([]);
  });

  it('returns events grouped by day for a month', () => {
    expect(helpers.getEventsForMonth(calendarEvents, 2025, 6)).toEqual({
      5: [{ title: 'Тренировка' }],
      24: [{ title: 'Питание' }, { title: 'Восстановление' }],
    });
    expect(helpers.getEventsForMonth(calendarEvents, 2025, 7)).toEqual({});
  });

  it('appends events to existing and missing dates', () => {
    const target = { '2025-07-05': [{ title: 'Первое' }] };
    helpers.addEventToDate(target, 2025, 6, 5, { title: 'Второе' });
    expect(target['2025-07-05']).toEqual([
      { title: 'Первое' },
      { title: 'Второе' },
    ]);
    helpers.addEventToDate(target, 2025, 6, 6, { title: 'Новое' });
    expect(target['2025-07-06']).toEqual([{ title: 'Новое' }]);
  });
});

describe('isDateAllowed', () => {
  it('rejects dates before the allowed range', () => {
    expect(helpers.isDateAllowed(2024, 11)).toBe(false);
    expect(helpers.isDateAllowed(MIN_YEAR, 0)).toBe(false);
    expect(helpers.isDateAllowed(MIN_YEAR, MIN_MONTH - 1)).toBe(false);
  });

  it('accepts dates inside the allowed range', () => {
    expect(helpers.isDateAllowed(MIN_YEAR, MIN_MONTH)).toBe(true);
    expect(helpers.isDateAllowed(MIN_YEAR, 11)).toBe(true);
    expect(helpers.isDateAllowed(MIN_YEAR + 1, 0)).toBe(true);
  });
});

describe('waterDateKey', () => {
  it('builds the current date key', () => {
    const now = new Date();
    const expected = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    expect(helpers.waterDateKey()).toBe(expected);
  });
});

describe('safeText', () => {
  it('escapes html sensitive characters', () => {
    expect(helpers.safeText(`&<>"'`)).toBe('&amp;&lt;&gt;&quot;&#39;');
    expect(helpers.safeText('<script>alert(1)</script>')).toBe(
      '&lt;script&gt;alert(1)&lt;/script&gt;'
    );
  });

  it('falls back to an empty string for nullish values', () => {
    expect(helpers.safeText(null)).toBe('');
    expect(helpers.safeText(undefined)).toBe('');
  });

  it('stringifies other values', () => {
    expect(helpers.safeText(42)).toBe('42');
    expect(helpers.safeText(true)).toBe('true');
  });
});

describe('formatRelativeDate', () => {
  it('labels today and tomorrow', () => {
    expect(helpers.formatRelativeDate(0, 'ru')).toBe('Сегодня');
    expect(helpers.formatRelativeDate(0, 'en')).toBe('Today');
    expect(helpers.formatRelativeDate(1, 'ru')).toBe('Завтра');
    expect(helpers.formatRelativeDate(1, 'en')).toBe('Tomorrow');
  });

  it('formats other days in russian', () => {
    const offset = 5;
    const expected = new Date();
    expected.setDate(expected.getDate() + offset);
    expect(helpers.formatRelativeDate(offset, 'ru')).toBe(
      `${expected.toLocaleDateString('ru-RU', { weekday: 'long' })} · ${expected.getDate()} ${getMonthName(expected.getMonth(), 'ru')}`
    );
  });

  it('formats other days in english', () => {
    const offset = 9;
    const expected = new Date();
    expected.setDate(expected.getDate() + offset);
    expect(helpers.formatRelativeDate(offset, 'en')).toBe(
      `${expected.toLocaleDateString('en-US', { weekday: 'long' })} · ${expected.getDate()} ${getMonthName(expected.getMonth(), 'en')}`
    );
  });

  it('defaults to russian', () => {
    const offset = 4;
    const expected = new Date();
    expected.setDate(expected.getDate() + offset);
    expect(helpers.formatRelativeDate(offset)).toBe(
      `${expected.toLocaleDateString('ru-RU', { weekday: 'long' })} · ${expected.getDate()} ${getMonthName(expected.getMonth(), 'ru')}`
    );
  });
});

describe('month and week ranges', () => {
  it('returns month boundaries', () => {
    expect(helpers.startOfMonth(new Date(2026, 9, 15))).toEqual(
      new Date(2026, 9, 1)
    );
    expect(helpers.endOfMonth(new Date(2026, 9, 15))).toEqual(
      new Date(2026, 9, 31)
    );
  });

  it('returns week boundaries for monday', () => {
    expect(helpers.startOfWeek(new Date(2026, 9, 12))).toEqual(
      new Date(2026, 9, 12)
    );
    expect(helpers.endOfWeek(new Date(2026, 9, 12))).toEqual(
      new Date(2026, 9, 18)
    );
  });

  it('returns week boundaries for sunday', () => {
    expect(helpers.startOfWeek(new Date(2026, 9, 11))).toEqual(
      new Date(2026, 9, 5)
    );
    expect(helpers.endOfWeek(new Date(2026, 9, 11))).toEqual(
      new Date(2026, 9, 11)
    );
  });

  it('adds days', () => {
    expect(helpers.addDays(new Date(2026, 9, 12), 5)).toEqual(
      new Date(2026, 9, 17)
    );
    expect(helpers.addDays(new Date(2026, 9, 12), -5)).toEqual(
      new Date(2026, 9, 7)
    );
    expect(helpers.addDays(new Date(2026, 9, 31), 1)).toEqual(
      new Date(2026, 10, 1)
    );
  });
});

describe('date comparisons', () => {
  it('compares days', () => {
    expect(
      helpers.isSameDay(new Date(2026, 9, 12), new Date(2026, 9, 12, 23, 59))
    ).toBe(true);
    expect(
      helpers.isSameDay(new Date(2026, 9, 12), new Date(2026, 9, 13))
    ).toBe(false);
    expect(
      helpers.isSameDay(new Date(2026, 9, 12), new Date(2026, 10, 12))
    ).toBe(false);
    expect(
      helpers.isSameDay(new Date(2026, 9, 12), new Date(2025, 9, 12))
    ).toBe(false);
  });

  it('compares months', () => {
    expect(
      helpers.isSameMonth(new Date(2026, 9, 1), new Date(2026, 9, 31))
    ).toBe(true);
    expect(
      helpers.isSameMonth(new Date(2026, 9, 31), new Date(2026, 10, 1))
    ).toBe(false);
    expect(
      helpers.isSameMonth(new Date(2026, 9, 12), new Date(2025, 9, 12))
    ).toBe(false);
  });
});

describe('format', () => {
  it('formats day, month and year', () => {
    expect(helpers.format(new Date(2026, 9, 12), 'd.M.yyyy')).toBe(
      '12.10.2026'
    );
    expect(helpers.format(new Date(2026, 0, 2), 'd.M.yyyy', 'en')).toBe(
      '2.1.2026'
    );
  });

  it('formats year first', () => {
    expect(helpers.format(new Date(2026, 9, 12), 'yyyy-M-d')).toBe(
      '2026-10-12'
    );
  });
});
