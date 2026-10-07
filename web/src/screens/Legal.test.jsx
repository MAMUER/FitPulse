import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Legal from './Legal';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'terms',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
  },
  t: (ru, _en) => ru,
  backFromLegal: vi.fn(),
};

function renderLegal(screen = 'terms') {
  useApp.mockReturnValue({
    ...defaultContext,
    state: { ...defaultContext.state, screen },
  });
  const path =
    screen === 'privacy'
      ? '/privacy'
      : screen === 'terms'
        ? '/terms'
        : '/consent';
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AppProvider>
        <Legal />
      </AppProvider>
    </MemoryRouter>
  );
}

describe('Legal', () => {
  it('renders terms page', () => {
    renderLegal('terms');
    expect(screen.getByText('Пользовательское соглашение')).toBeInTheDocument();
  });

  it('renders privacy page', () => {
    renderLegal('privacy');
    expect(screen.getByText('Политика конфиденциальности')).toBeInTheDocument();
  });

  it('renders consent page', () => {
    renderLegal('consent');
    expect(
      screen.getByText('Соглашение об использовании персональных данных')
    ).toBeInTheDocument();
  });

  it('calls backFromLegal on close', () => {
    renderLegal();
    screen.getByText('Назад').click();
    expect(defaultContext.backFromLegal).toHaveBeenCalled();
  });
});
