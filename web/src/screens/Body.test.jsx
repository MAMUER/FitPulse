import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Body from './Body';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'body',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    height: 175,
    weight: 74.2,
  },
  t: (ru, _en) => ru,
  logWeight: vi.fn(),
  addWeight: vi.fn(),
  openMetric: vi.fn(),
  go: vi.fn(),
};

function renderBody() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <AppProvider>
      <Body />
    </AppProvider>
  );
}

describe('Body', () => {
  it('renders body profile screen', () => {
    renderBody();
    expect(screen.getByText('Профиль тела')).toBeInTheDocument();
  });
});
