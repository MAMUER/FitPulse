import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import { getDateKey } from '../utils/helpers';
import Calendar from './Calendar';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

// The calendar grid is driven by state.calendarYear / calendarMonthIndex.
const YEAR = 2026;
const MONTH = 4; // May, 0-based

function makeContext(overrides = {}) {
  const { state: stateOverrides, ...rest } = overrides;
  return {
    state: {
      language: 'ru',
      calendarYear: YEAR,
      calendarMonthIndex: MONTH,
      calendarEvents: {},
      ...stateOverrides,
    },
    t: (ru, _en) => ru,
    getMonthName: (_monthIndex, _language) => 'Май',
    loadCalendarEvents: vi.fn(),
    changeCalendar: vi.fn(),
    quickAddEvent: vi.fn(),
    openCalendarPage: vi.fn(),
    closeCalendarPage: vi.fn(),
    selectCalendarDay: vi.fn(),
    closeDateModal: vi.fn(),
    addEvent: vi.fn(),
    deleteCalendarEvent: vi.fn().mockResolvedValue({}),
    ...rest,
  };
}

describe('Calendar', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('loads events and shows the day grid', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Calendar />);

    expect(context.loadCalendarEvents).toHaveBeenCalled();
    // The page-title eyebrow is hardcoded ("PLANNER").
    expect(screen.getByText('PLANNER')).toBeInTheDocument();
    // The card view renders a mini month grid: one button per day. The
    // "N Month" aria-labels belong to the full calendar page view, not this one.
    const dayButtons = document.querySelectorAll('.month button');
    expect(dayButtons.length).toBeGreaterThanOrEqual(28);
    expect(dayButtons[0].textContent).toContain('1');
  });

  it('adds event from the day modal', () => {
    const context = makeContext({
      state: { showDateModal: true, dateModalDate: 1 },
    });
    useApp.mockReturnValue(context);

    render(<Calendar />);
    fireEvent.change(screen.getByLabelText('Название'), {
      target: { value: 'Тест' },
    });
    fireEvent.change(screen.getByLabelText('Тип'), {
      target: { value: 'food' },
    });
    fireEvent.change(screen.getByLabelText('Время'), {
      target: { value: '10:00' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Добавить событие/ }));

    expect(context.addEvent).toHaveBeenCalled();
  });

  it('deletes event', async () => {
    const context = makeContext({
      state: {
        showDateModal: true,
        dateModalDate: 1,
        calendarEvents: {
          [getDateKey(YEAR, MONTH, 1)]: [
            {
              id: 123,
              title: 'Тренировка',
              type: 'training',
              time: '09:00',
              description: 'test',
            },
          ],
        },
      },
    });
    useApp.mockReturnValue(context);

    render(<Calendar />);
    fireEvent.click(screen.getByRole('button', { name: /Удалить/ }));

    await waitFor(() => expect(context.deleteCalendarEvent).toHaveBeenCalled());
  });

  it('renders the day modal with type and time selectors', () => {
    const context = makeContext({
      state: { showDateModal: true, dateModalDate: 1 },
    });
    useApp.mockReturnValue(context);

    render(<Calendar />);

    expect(screen.getByLabelText('Название')).toBeInTheDocument();
    expect(screen.getByLabelText('Тип')).toBeInTheDocument();
    expect(screen.getByLabelText('Время')).toBeInTheDocument();
    expect(screen.getByText('Нет событий на эту дату')).toBeInTheDocument();
  });

  it('renders events with fallback fields and deletes', async () => {
    const context = makeContext({
      state: {
        showDateModal: true,
        dateModalDate: 1,
        calendarEvents: {
          [getDateKey(YEAR, MONTH, 1)]: [
            { title: 'Событие', type: 'unknown', description: '' },
          ],
        },
      },
    });
    useApp.mockReturnValue(context);

    render(<Calendar />);
    expect(screen.getByText('Событие')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Удалить/ }));

    await waitFor(() => expect(context.deleteCalendarEvent).toHaveBeenCalled());
  });

  it('handles null calendarEventsList', () => {
    const context = makeContext({ state: { calendarEvents: undefined } });
    useApp.mockReturnValue(context);

    render(<Calendar />);

    expect(context.loadCalendarEvents).toHaveBeenCalled();
    // The page-title eyebrow is hardcoded ("PLANNER"), the planner header below
    // the month controls is translated.
    expect(screen.getByText('PLANNER')).toBeInTheDocument();
  });

  it('renders events with missing id type and description', async () => {
    const context = makeContext({
      state: {
        showDateModal: true,
        dateModalDate: 1,
        calendarEvents: {
          [getDateKey(YEAR, MONTH, 1)]: [
            {
              id: null,
              title: 'Событие',
              type: 'unknown_type',
              time: null,
              description: null,
            },
          ],
        },
      },
    });
    useApp.mockReturnValue(context);

    render(<Calendar />);
    expect(screen.getByText('Событие')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Удалить/ }));
    await waitFor(() => expect(context.deleteCalendarEvent).toHaveBeenCalled());
  });

  it('deletes event on matching day', async () => {
    const context = makeContext({
      state: {
        showDateModal: true,
        dateModalDate: 1,
        calendarEvents: {
          [getDateKey(YEAR, MONTH, 1)]: [
            { id: 7, title: 'E', type: 'training', time: '09:00' },
          ],
        },
      },
    });
    useApp.mockReturnValue(context);

    render(<Calendar />);
    fireEvent.click(screen.getByRole('button', { name: /Удалить/ }));

    await waitFor(() => expect(context.deleteCalendarEvent).toHaveBeenCalled());
    expect(context.deleteCalendarEvent).toHaveBeenCalledWith(
      getDateKey(YEAR, MONTH, 1),
      0
    );
  });
});
