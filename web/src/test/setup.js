import { vi } from 'vitest';
import '@testing-library/jest-dom';
import React from 'react';

global.React = React;

Object.defineProperty(window, 'localStorage', {
  value: {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
  },
  writable: true,
});

Object.defineProperty(window, 'prompt', {
  value: vi.fn(),
  writable: true,
});

const originalLocation = window.location;

Object.defineProperty(window, 'location', {
  value: {
    ...originalLocation,
    assign: vi.fn(),
    replace: vi.fn(),
    reload: vi.fn(),
  },
  writable: true,
});

vi.mock('react-chartjs-2', () => ({
  Chart: () => null,
  Doughnut: () => null,
  Line: () => null,
  Bar: () => null,
}));
