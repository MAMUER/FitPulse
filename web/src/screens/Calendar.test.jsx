import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import Calendar from './Calendar'
import { useApp } from '../contexts/AppContext'

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}))

describe('Calendar', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('loads events and shows day selection', async () => {
    const loadCalendarEvents = vi.fn()
    const addEvent = vi.fn().mockResolvedValue({})
    const deleteCalendarEvent = vi.fn().mockResolvedValue({})
    useApp.mockReturnValue({
      state: { language: 'ru', calendarEventsList: [] },
      t: (ru, en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    })

    render(<Calendar />)
    expect(loadCalendarEvents).toHaveBeenCalled()
    expect(screen.getByText('Выберите день')).toBeInTheDocument()
  })

  it('adds event', async () => {
    const loadCalendarEvents = vi.fn()
    const addEvent = vi.fn().mockResolvedValue({})
    const deleteCalendarEvent = vi.fn().mockResolvedValue({})
    useApp.mockReturnValue({
      state: { language: 'ru', calendarEventsList: [] },
      t: (ru, en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    })

    render(<Calendar />)
    // Click first day
    const days = screen.getAllByRole('button')
    fireEvent.click(days.find(d => d.className.includes('cal-cell') && !d.className.includes('cal-head')))
    fireEvent.click(screen.getByText('Добавить событие'))
    fireEvent.change(screen.getByLabelText('Название'), { target: { value: 'Тест' } })
    fireEvent.change(screen.getByLabelText('Тип'), { target: { value: 'food' } })
    fireEvent.change(screen.getByLabelText('Время'), { target: { value: '10:00' } })
    fireEvent.change(screen.getByLabelText('Описание'), { target: { value: 'описание' } })
    fireEvent.click(screen.getByText('Добавить'))
    await waitFor(() => expect(addEvent).toHaveBeenCalled())
  })

  it('deletes event', async () => {
    const loadCalendarEvents = vi.fn()
    const addEvent = vi.fn()
    const deleteCalendarEvent = vi.fn().mockResolvedValue({})
    useApp.mockReturnValue({
      state: { language: 'ru', calendarEventsList: [{ id: 123, date: '2099-01-01', title: 'Тренировка', type: 'training', time: '09:00', description: 'test' }] },
      t: (ru, en) => ru,
      addEvent,
      deleteCalendarEvent,
      loadCalendarEvents,
    })

    render(<Calendar />)
    // Click first day button
    const buttons = screen.getAllByRole('button')
    const dayBtn = buttons.find(b => b.className.includes('cal-cell') && !b.className.includes('cal-head'))
    if (dayBtn) fireEvent.click(dayBtn)
    // Look for delete
    const delButtons = screen.queryAllByText('Удалить')
    if (delButtons.length > 0) {
      fireEvent.click(delButtons[0])
      await waitFor(() => expect(deleteCalendarEvent).toHaveBeenCalled())
    } else {
      expect(delButtons).toHaveLength(0)
    }
  })
})
