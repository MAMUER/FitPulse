import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import AI from './AI';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'ai',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    messages: [],
  },
  t: (ru, _en) => ru,
  sendAI: vi.fn(),
  quickAI: vi.fn(),
  askAI: vi.fn(),
  go: vi.fn(),
};

function renderAI() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <AppProvider>
      <AI />
    </AppProvider>
  );
}

describe('AI', () => {
  it('renders AI screen', () => {
    renderAI();
    expect(screen.getByText('AI-советник')).toBeInTheDocument();
  });
});
