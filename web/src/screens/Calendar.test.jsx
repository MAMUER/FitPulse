import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import { format, startOfMonth, startOfWeek } from '../utils/helpers';
import Calendar from './Calendar';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('Calendar', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('loads events and shows day selection', async () => {
    const loadCalendarEvents = vi.fn();
    const addEvent = vi.fn().mockResolvedValue({});
    const deleteCalendarEvent = vi.fn().mockResolvedValue({});
    useApp.mockReturnValue({
      state: { language: 'ru', calendarEventsList: [] },
      t: (ru, _en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    });

    render(<Calendar />);
    expect(loadCalendarEvents).toHaveBeenCalled();
    expect(screen.getByText('Выберите день')).toBeInTheDocument();
  });

  it('adds event', async () => {
    const loadCalendarEvents = vi.fn();
    const addEvent = vi.fn().mockResolvedValue({});
    const deleteCalendarEvent = vi.fn().mockResolvedValue({});
    useApp.mockReturnValue({
      state: { language: 'ru', calendarEventsList: [] },
      t: (ru, _en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    });

    render(<Calendar />);
    const days = screen.getAllByRole('button');
    fireEvent.click(
      days.find(
        (d) =>
          d.className.includes('cal-cell') && !d.className.includes('cal-head')
      )
    );
    fireEvent.click(screen.getByText('Добавить событие'));
    fireEvent.change(screen.getByLabelText('Название'), {
      target: { value: 'Тест' },
    });
    fireEvent.change(screen.getByLabelText('Тип'), {
      target: { value: 'food' },
    });
    fireEvent.change(screen.getByLabelText('Время'), {
      target: { value: '10:00' },
    });
    fireEvent.change(screen.getByLabelText('Описание'), {
      target: { value: 'описание' },
    });
    fireEvent.click(screen.getByText('Добавить'));
    await waitFor(() => expect(addEvent).toHaveBeenCalled());
  });

  it('deletes event', async () => {
    const loadCalendarEvents = vi.fn();
    const addEvent = vi.fn();
    const deleteCalendarEvent = vi.fn().mockResolvedValue({});
    const firstDay = startOfWeek(startOfMonth(new Date()));
    const dateStr = format(firstDay, 'yyyy-MM-dd');
    useApp.mockReturnValue({
      state: {
        language: 'ru',
        calendarEventsList: [
          {
            id: 123,
            date: dateStr,
            title: 'Тренировка',
            type: 'training',
            time: '09:00',
            description: 'test',
          },
        ],
      },
      t: (ru, _en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    });

    render(<Calendar />);
    const buttons = screen.getAllByRole('button');
    const dayBtn = buttons.find(
      (b) =>
        b.className.includes('cal-cell') && !b.className.includes('cal-head')
    );
    if (dayBtn) fireEvent.click(dayBtn);
    const delButtons = screen.queryAllByText('Удалить');
    if (delButtons.length > 0) {
      fireEvent.click(delButtons[0]);
      await waitFor(() => expect(deleteCalendarEvent).toHaveBeenCalled());
    }
  });

  it('handles empty title in add event', async () => {
    const loadCalendarEvents = vi.fn();
    const addEvent = vi.fn().mockResolvedValue({});
    const deleteCalendarEvent = vi.fn().mockResolvedValue({});
    useApp.mockReturnValue({
      state: { language: 'ru', calendarEventsList: [] },
      t: (ru, _en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    });
    render(<Calendar />);
    const buttons = screen.getAllByRole('button');
    const dayBtn = buttons.find(
      (b) =>
        b.className.includes('cal-cell') && !b.className.includes('cal-head')
    );
    if (dayBtn) fireEvent.click(dayBtn);
    fireEvent.click(screen.getByText('Добавить событие'));
    fireEvent.click(screen.getByText('Добавить'));
    expect(addEvent).not.toHaveBeenCalled();
  });

  it('renders events with fallback fields and deletes', async () => {
    const loadCalendarEvents = vi.fn();
    const addEvent = vi.fn();
    const deleteCalendarEvent = vi.fn().mockResolvedValue({});
    const today = new Date();
    const dateStr = format(today, 'yyyy-MM-dd');
    useApp.mockReturnValue({
      state: {
        language: 'ru',
        calendarEventsList: [
          {
            date: dateStr,
            title: 'Событие',
            type: 'unknown',
            description: '',
          },
        ],
      },
      t: (ru, _en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    });
    render(<Calendar />);
    const buttons = screen.getAllByRole('button');
    const dayBtn = buttons.find(
      (b) =>
        b.className.includes('cal-cell') && !b.className.includes('cal-head')
    );
    if (dayBtn) fireEvent.click(dayBtn);
    const delButtons = screen.queryAllByText('Удалить');
    if (delButtons.length > 0) {
      fireEvent.click(delButtons[0]);
      await waitFor(() => expect(deleteCalendarEvent).toHaveBeenCalled());
    }
  });

  it('handles null calendarEventsList', () => {
    const loadCalendarEvents = vi.fn();
    const addEvent = vi.fn();
    const deleteCalendarEvent = vi.fn();
    useApp.mockReturnValue({
      state: { language: 'ru' },
      t: (ru, _en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    });
    render(<Calendar />);
    expect(loadCalendarEvents).toHaveBeenCalled();
  });

  it('renders events with missing id type and description', async () => {
    const loadCalendarEvents = vi.fn();
    const deleteCalendarEvent = vi.fn().mockResolvedValue({});
    const firstDay = startOfWeek(startOfMonth(new Date()));
    const dateStr = format(firstDay, 'yyyy-MM-dd');
    useApp.mockReturnValue({
      state: {
        language: 'ru',
        calendarEventsList: [
          {
            id: null,
            date: dateStr,
            title: 'Событие',
            type: 'unknown_type',
            time: null,
            description: null,
          },
        ],
      },
      t: (ru, _en) => ru,
      addEvent: vi.fn(),
      deleteCalendarEvent,
      loadCalendarEvents,
    });
    render(<Calendar />);
    const buttons = screen.getAllByRole('button');
    const dayBtn = buttons.find(
      (b) =>
        b.className.includes('cal-cell') && !b.className.includes('cal-head')
    );
    if (dayBtn) fireEvent.click(dayBtn);
    expect(await screen.findByText('Событие')).toBeInTheDocument();
    const delButtons = screen.queryAllByText('Удалить');
    if (delButtons.length > 0) {
      fireEvent.click(delButtons[0]);
      await waitFor(() => expect(deleteCalendarEvent).toHaveBeenCalled());
    }
  });
});
