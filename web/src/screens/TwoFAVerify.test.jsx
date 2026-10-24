import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import TwoFAVerify from './TwoFAVerify'
import { useApp } from '../contexts/AppContext'

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}))

describe('TwoFAVerify', () => {
  beforeEach(() => {
    delete window.location
    window.location = { search: '?temp_token=abc123' }
  })

  it('submits verification', async () => {
    const verify2FA = vi.fn().mockResolvedValue({})
    useApp.mockReturnValue({
      verify2FA,
      t: (ru, en) => ru,
    })

    render(<TwoFAVerify />)
    fireEvent.change(screen.getByLabelText('Код'), { target: { value: '654321' } })
    fireEvent.click(screen.getByText('Это резервный код'))
    fireEvent.submit(screen.getByRole('button', { name: 'Подтвердить' }).closest('form'))
    await waitFor(() => expect(verify2FA).toHaveBeenCalled())
    expect(verify2FA).toHaveBeenCalledWith('abc123', '654321', true)
  })
})
