import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Calendar from './Calendar';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'calendar',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    calendarMonthIndex: new Date().getMonth(),
    calendarYear: new Date().getFullYear(),
    calendarEvents: {},
    calendarEventsList: [],
  },
  t: (ru, _en) => ru,
  addEvent: vi.fn(),
  deleteCalendarEvent: vi.fn(),
  loadCalendarEvents: vi.fn(),
  selectCalendarDay: vi.fn(),
  closeDateModal: vi.fn(),
  addEventToDay: vi.fn(),
  changeCalendar: vi.fn(),
};

function renderCalendar() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Calendar />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Calendar', () => {
  it('renders calendar screen', () => {
    renderCalendar();
    expect(screen.getByText('Календарь')).toBeInTheDocument();
  });
});
