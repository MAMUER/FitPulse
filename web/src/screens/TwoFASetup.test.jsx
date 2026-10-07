import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import TwoFASetup from './TwoFASetup';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'twofa-setup',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
  },
  t: (ru, _en) => ru,
  confirm2FA: vi.fn(),
};

function renderTwoFASetup() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <TwoFASetup />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('TwoFASetup', () => {
  it('renders 2FA setup screen', () => {
    renderTwoFASetup();
    expect(screen.getByText('Настройка 2FA')).toBeInTheDocument();
  });

  it('renders QR and input fields', () => {
    renderTwoFASetup();
    expect(screen.getByText('Секрет')).toBeInTheDocument();
    expect(screen.getByText('Код подтверждения')).toBeInTheDocument();
  });

  it('renders confirm button', () => {
    renderTwoFASetup();
    expect(screen.getByText('Подтвердить')).toBeInTheDocument();
  });
});
