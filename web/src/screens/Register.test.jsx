import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import Register from './Register'
import { useApp } from '../contexts/AppContext'

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}))

vi.mock('../components/AuthLayout', () => ({
  default: ({ children }) => <div>{children}</div>,
}))

describe('Register', () => {
  it('navigates to legal and logs in', () => {
    const doRegister = vi.fn()
    const socialLogin = vi.fn()
    const go = vi.fn()
    useApp.mockReturnValue({ t: (ru, en) => ru, doRegister, socialLogin, go })

    render(<Register />)
    fireEvent.click(screen.getByText('Соглашение'))
    expect(go).toHaveBeenCalledWith('legal', 'terms')
    fireEvent.click(screen.getByText('Персональные данные'))
    expect(go).toHaveBeenCalledWith('legal', 'consent')
    fireEvent.click(screen.getByText('Политика'))
    expect(go).toHaveBeenCalledWith('legal', 'privacy')
    fireEvent.click(screen.getByText('Зарегистрироваться'))
    expect(doRegister).toHaveBeenCalled()
    fireEvent.click(screen.getByText('Войти'))
    expect(go).toHaveBeenCalledWith('login')
    fireEvent.click(screen.getByText(/Яндекс/))
    expect(socialLogin).toHaveBeenCalledWith('yandex')
  })
})
