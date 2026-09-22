import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Profile from './Profile';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'profile',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    profile: { email: 'mih@example.com', twoFactor: false },
    profileLoaded: true,
  },
  t: (ru, _en) => ru,
  logout: vi.fn(),
  go: vi.fn(),
  toggleLanguage: vi.fn(),
  toggleTheme: vi.fn(),
  toggleHighContrast: vi.fn(),
  load2FAStatus: vi.fn(),
  setup2FA: vi.fn(),
  disable2FA: vi.fn(),
  loadProfile: vi.fn(),
  saveProfile: vi.fn(),
};

function renderProfile() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Profile />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Profile', () => {
  it('renders profile page', () => {
    renderProfile();
    expect(screen.getByText('Профиль')).toBeInTheDocument();
    expect(screen.getByText('mih@example.com')).toBeInTheDocument();
  });

  it('toggles theme', () => {
    renderProfile();
    const ctx = useApp();
    const buttons = screen.getAllByText('Сменить');
    buttons[1].click();
    expect(ctx.toggleTheme).toHaveBeenCalled();
  });

  it('logs out', () => {
    renderProfile();
    const ctx = useApp();
    screen.getByText('Выйти').click();
    expect(ctx.logout).toHaveBeenCalled();
  });
});
