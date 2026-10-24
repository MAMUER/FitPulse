import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import Nutrition from './Nutrition'
import { useApp } from '../contexts/AppContext'

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}))

describe('Nutrition', () => {
  it('adds and removes meals', async () => {
    const loadMeals = vi.fn()
    const createMeal = vi.fn().mockResolvedValue({})
    const removeMeal = vi.fn().mockResolvedValue({})
    const go = vi.fn()
    useApp.mockReturnValue({
      state: { mealsList: [{ id: 1, name: 'Яйца', calories: 200, time: '08:00' }], calorieGoal: 2000 },
      t: (ru, en) => ru,
      go,
      loadMeals,
      createMeal,
      removeMeal,
    })

    render(<Nutrition />)
    expect(loadMeals).toHaveBeenCalled()
    fireEvent.click(screen.getByText('Удалить'))
    await waitFor(() => expect(removeMeal).toHaveBeenCalledWith(1))
    fireEvent.change(screen.getByLabelText('Название'), { target: { value: 'Рис' } })
    fireEvent.change(screen.getByLabelText('Калории'), { target: { value: '300' } })
    fireEvent.change(screen.getByLabelText('Время'), { target: { value: '12:00' } })
    fireEvent.click(screen.getByText('Добавить блюдо'))
    await waitFor(() => expect(createMeal).toHaveBeenCalled())
    fireEvent.click(screen.getByText('На главную'))
    expect(go).toHaveBeenCalledWith('home')
  })

  it('adds meal with defaults', async () => {
    const loadMeals = vi.fn()
    const createMeal = vi.fn().mockResolvedValue({})
    useApp.mockReturnValue({
      state: { mealsList: [] },
      t: (ru, en) => ru,
      go: vi.fn(),
      loadMeals,
      createMeal,
      removeMeal: vi.fn(),
    })

    render(<Nutrition />)
    fireEvent.click(screen.getByText('Добавить блюдо'))
    await waitFor(() => expect(createMeal).toHaveBeenCalled())
  })
})
