import { vi } from 'vitest'
import '@testing-library/jest-dom'
import React from 'react'

global.React = React

Object.defineProperty(window, 'localStorage', {
  value: {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn()
  },
  writable: true
})

vi.mock('react-chartjs-2', () => ({
  Chart: () => null,
  Doughnut: () => null,
  Line: () => null,
  Bar: () => null
}))