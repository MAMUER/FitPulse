import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import Body from './Body'
import { useApp } from '../contexts/AppContext'

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}))

describe('Body', () => {
  it('loads and saves body composition', async () => {
    const loadBodyComposition = vi.fn()
    const saveBodyComposition = vi.fn().mockResolvedValue({})
    const go = vi.fn()
    useApp.mockReturnValue({
      state: { height: 175, weight: 75, bodyComposition: [{ recorded_at: '2026-01-01T00:00:00Z', weight_kg: 75, bmi: 24.5 }] },
      t: (ru, en) => ru,
      go,
      loadBodyComposition,
      saveBodyComposition,
    })

    render(<Body />)
    expect(loadBodyComposition).toHaveBeenCalled()
    fireEvent.change(screen.getByLabelText('Жир, %'), { target: { value: '18' } })
    fireEvent.change(screen.getByLabelText('Мышцы, %'), { target: { value: '35' } })
    fireEvent.change(screen.getByLabelText('Вода, %'), { target: { value: '50' } })
    fireEvent.click(screen.getByText('Сохранить'))
    await waitFor(() => expect(saveBodyComposition).toHaveBeenCalled())
    fireEvent.click(screen.getByText('На главную'))
    expect(go).toHaveBeenCalledWith('home')
  })
})
