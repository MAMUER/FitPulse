import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import CookieConsent from './CookieConsent'
import { useApp } from '../contexts/AppContext'

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}))

describe('CookieConsent', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    localStorage.clear()
  })

  it('shows when not accepted', async () => {
    useApp.mockReturnValue({ t: (ru, en) => ru })
    localStorage.removeItem('cookie-consent')
    render(<CookieConsent />)
    await screen.findByText('Принять')
    fireEvent.click(screen.getByText('Принять'))
    expect(localStorage.getItem('cookie-consent')).toBe('accepted')
  })

  it('shows when declined not stored', async () => {
    useApp.mockReturnValue({ t: (ru, en) => en })
    localStorage.removeItem('cookie-consent')
    render(<CookieConsent />)
    await screen.findByText('Decline')
    fireEvent.click(screen.getByText('Decline'))
    expect(localStorage.getItem('cookie-consent')).toBe('declined')
  })

  it('does not show when accepted', () => {
    useApp.mockReturnValue({ t: (ru, en) => ru })
    localStorage.setItem('cookie-consent', 'accepted')
    const { container } = render(<CookieConsent />)
    expect(container.firstChild).toBeNull()
  })
})
