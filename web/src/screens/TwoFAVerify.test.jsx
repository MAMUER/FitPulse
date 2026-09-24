import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import TwoFAVerify from './TwoFAVerify';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'twofa-verify',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
  },
  t: (ru, _en) => ru,
  verify2FA: vi.fn(),
};

function renderTwoFAVerify() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <TwoFAVerify />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('TwoFAVerify', () => {
  it('renders 2FA verify screen', () => {
    renderTwoFAVerify();
    expect(
      screen.getByText('Двухфакторная аутентификация')
    ).toBeInTheDocument();
  });

  it('renders code input and checkbox', () => {
    renderTwoFAVerify();
    expect(screen.getByText('Код')).toBeInTheDocument();
    expect(screen.getByText('Это резервный код')).toBeInTheDocument();
  });

  it('renders verify button', () => {
    renderTwoFAVerify();
    expect(screen.getByText('Подтвердить')).toBeInTheDocument();
  });
});
