import { render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import Confirm from './Confirm'
import { useApp } from '../contexts/AppContext'
import { MemoryRouter } from 'react-router-dom'

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}))

describe('Confirm', () => {
  it('confirms email successfully', async () => {
    const confirmEmail = vi.fn().mockResolvedValue({ status: 'ok' })
    useApp.mockReturnValue({ t: (ru, en) => ru, confirmEmail })
    render(
      <MemoryRouter initialEntries={['/confirm?token=abc']}>
        <Confirm />
      </MemoryRouter>
    )
    await waitFor(() => expect(confirmEmail).toHaveBeenCalledWith('abc'))
    await waitFor(() => expect(screen.getAllByText('Email подтверждён').length).toBeGreaterThan(0))
  })

  it('shows error when token missing', async () => {
    const confirmEmail = vi.fn()
    useApp.mockReturnValue({ t: (ru, en) => ru, confirmEmail })
    render(
      <MemoryRouter initialEntries={['/confirm']}>
        <Confirm />
      </MemoryRouter>
    )
    await screen.findByText('Токен не указан')
    expect(confirmEmail).not.toHaveBeenCalled()
  })

  it('shows error when confirmation fails', async () => {
    const confirmEmail = vi.fn().mockRejectedValue(new Error('boom'))
    useApp.mockReturnValue({ t: (ru, en) => en, confirmEmail })
    render(
      <MemoryRouter initialEntries={['/confirm?token=bad']}>
        <Confirm />
      </MemoryRouter>
    )
    await waitFor(() => expect(confirmEmail).toHaveBeenCalledWith('bad'))
    await waitFor(() => {
      const texts = screen.getAllByText('Confirmation error')
      expect(texts.length).toBeGreaterThan(0)
    })
  })

  it('shows message from response', async () => {
    const confirmEmail = vi.fn().mockResolvedValue({ status: 'error', message: 'bad token' })
    useApp.mockReturnValue({ t: (ru, en) => ru, confirmEmail })
    render(
      <MemoryRouter initialEntries={['/confirm?token=bad']}>
        <Confirm />
      </MemoryRouter>
    )
    await waitFor(() => expect(confirmEmail).toHaveBeenCalledWith('bad'))
    expect(await screen.findByText('bad token')).toBeInTheDocument()
  })
})
