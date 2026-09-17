import { render, screen } from '@testing-library/react';
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
    currentMonth: new Date(),
    selectedDays: [],
  },
  t: (ru, _en) => ru,
  changeCalendar: vi.fn(),
  quickAddEvent: vi.fn(),
  openCalendarPage: vi.fn(),
  toggleDay: vi.fn(),
};

function renderCalendar() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <AppProvider>
      <Calendar />
    </AppProvider>
  );
}

describe('Calendar', () => {
  it('renders calendar screen', () => {
    renderCalendar();
    expect(screen.getByText('Календарь')).toBeInTheDocument();
  });
});
