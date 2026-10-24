import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Profile from './Profile';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const baseContext = (overrides = {}) => ({
  state: {
    screen: 'profile',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    profile: {
      email: 'mih@example.com',
      twoFactor: false,
      name: 'Михаил',
      gender: 'male',
      phone: '+79001234567',
      bio: 'Hello',
      status: 'active',
    },
    ...overrides.state,
  },
  t: (ru, _en) => ru,
  logout: vi.fn(),
  go: vi.fn(),
  toggleLanguage: vi.fn(),
  toggleTheme: vi.fn(),
  toggleHighContrast: vi.fn(),
  load2FAStatus: vi.fn(),
  setup2FA: vi.fn(),
  disable2FA: vi.fn().mockResolvedValue({}),
  loadProfile: vi.fn(),
  saveProfile: vi.fn().mockResolvedValue({}),
  ...overrides,
});

function renderProfile(context) {
  useApp.mockReturnValue(context);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Profile />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Profile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders profile page', () => {
    renderProfile(baseContext());
    expect(screen.getByText('Профиль')).toBeInTheDocument();
    expect(screen.getByText('mih@example.com')).toBeInTheDocument();
  });

  it('loads profile and 2FA status on mount', () => {
    const loadProfile = vi.fn();
    const load2FAStatus = vi.fn();
    renderProfile(baseContext({ loadProfile, load2FAStatus }));
    expect(loadProfile).toHaveBeenCalledTimes(1);
    expect(load2FAStatus).toHaveBeenCalledTimes(1);
  });

  it('toggles language', async () => {
    const toggleLanguage = vi.fn();
    const user = userEvent.setup();
    renderProfile(baseContext({ toggleLanguage }));
    const buttons = screen.getAllByText('Сменить');
    await user.click(buttons[0]);
    expect(toggleLanguage).toHaveBeenCalled();
  });

  it('toggles theme', async () => {
    const toggleTheme = vi.fn();
    const user = userEvent.setup();
    renderProfile(baseContext({ toggleTheme }));
    const buttons = screen.getAllByText('Сменить');
    await user.click(buttons[1]);
    expect(toggleTheme).toHaveBeenCalled();
  });

  it('toggles high contrast', async () => {
    const toggleHighContrast = vi.fn();
    const user = userEvent.setup();
    renderProfile(baseContext({ toggleHighContrast }));
    const buttons = screen.getAllByText('Сменить');
    await user.click(buttons[2]);
    expect(toggleHighContrast).toHaveBeenCalled();
  });

  it('logs out', () => {
    const logout = vi.fn();
    renderProfile(baseContext({ logout }));
    screen.getByText('Выйти').click();
    expect(logout).toHaveBeenCalled();
  });

  it('navigates home', () => {
    const go = vi.fn();
    renderProfile(baseContext({ go }));
    screen.getByText('На главную').click();
    expect(go).toHaveBeenCalledWith('home');
  });

  it('saves profile', async () => {
    const user = userEvent.setup();
    const saveProfile = vi.fn().mockResolvedValue({});
    renderProfile(baseContext({ saveProfile }));
    await user.click(screen.getByText('Сохранить профиль'));
    await waitFor(() => {
      expect(saveProfile).toHaveBeenCalledWith({
        full_name: 'Михаил',
        gender: 'male',
        phone: '+79001234567',
        bio: 'Hello',
        status: 'active',
      });
    });
  });

  it('shows saving state while saving profile', async () => {
    const user = userEvent.setup();
    let resolveSave;
    const saveProfile = vi.fn(
      () =>
        new Promise((resolve) => {
          resolveSave = resolve;
        })
    );
    renderProfile(baseContext({ saveProfile }));
    await user.click(screen.getByText('Сохранить профиль'));
    expect(screen.getByText('Сохранение...')).toBeInTheDocument();
    resolveSave({});
    await waitFor(() => {
      expect(screen.queryByText('Сохранение...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Сохранить профиль')).toBeInTheDocument();
  });

  it('shows language subtitle RU', () => {
    renderProfile(baseContext());
    expect(screen.getByText('RU')).toBeInTheDocument();
  });

  it('shows theme subtitle dark', () => {
    renderProfile(baseContext());
    expect(screen.getByText('dark')).toBeInTheDocument();
  });

  it('shows high contrast subtitle OFF', () => {
    renderProfile(baseContext({ state: { highContrast: false } }));
    expect(screen.getAllByText('OFF').length).toBeGreaterThanOrEqual(1);
  });

  it('shows enable 2FA button when twoFactor is false', () => {
    renderProfile(baseContext({ state: { profile: { twoFactor: false } } }));
    expect(screen.getByText('Включить')).toBeInTheDocument();
    expect(screen.queryByText('Отключить')).not.toBeInTheDocument();
  });

  it('shows disable 2FA button when twoFactor is true', () => {
    renderProfile(baseContext({ state: { profile: { twoFactor: true } } }));
    expect(screen.getByText('Отключить')).toBeInTheDocument();
    expect(screen.queryByText('Включить')).not.toBeInTheDocument();
  });

  it('shows code input when twoFactor is true', () => {
    renderProfile(baseContext({ state: { profile: { twoFactor: true } } }));
    expect(screen.getByLabelText('Код для отключения')).toBeInTheDocument();
    expect(screen.getByText('Код для отключения')).toBeInTheDocument();
  });

  it('does not show code input when twoFactor is false', () => {
    renderProfile(baseContext({ state: { profile: { twoFactor: false } } }));
    expect(screen.queryByLabelText('Код для отключения')).not.toBeInTheDocument();
  });

  it('sets up 2FA when clicking Enable', async () => {
    const user = userEvent.setup();
    const setup2FA = vi.fn();
    renderProfile(baseContext({ state: { profile: { twoFactor: false } }, setup2FA }));
    await user.click(screen.getByText('Включить'));
    expect(setup2FA).toHaveBeenCalled();
  });

  it('disables 2FA when clicking Disable with code', async () => {
    const user = userEvent.setup();
    const disable2FA = vi.fn().mockResolvedValue({});
    renderProfile(
      baseContext({
        state: { profile: { twoFactor: true } },
        disable2FA,
      })
    );
    await user.type(screen.getByLabelText('Код для отключения'), '123456');
    await user.click(screen.getByText('Отключить'));
    await waitFor(() => {
      expect(disable2FA).toHaveBeenCalledWith('123456');
    });
  });

  it('shows disabling state while disabling 2FA', async () => {
    const user = userEvent.setup();
    let resolveDisable;
    const disable2FA = vi.fn(
      () =>
        new Promise((resolve) => {
          resolveDisable = resolve;
        })
    );
    renderProfile(
      baseContext({
        state: { profile: { twoFactor: true } },
        disable2FA,
      })
    );
    await user.type(screen.getByLabelText('Код для отключения'), '123456');
    await user.click(screen.getByText('Отключить'));
    expect(screen.getByText('Отключение...')).toBeInTheDocument();
    resolveDisable({});
    await waitFor(() => {
      expect(screen.queryByText('Отключение...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Отключить')).toBeInTheDocument();
  });

  it('uses English translations when language is en', () => {
    const enContext = baseContext({
      state: { language: 'en', profile: { twoFactor: true } },
      t: (_ru, en) => en,
    });
    renderProfile(enContext);
    expect(screen.getByText('Profile')).toBeInTheDocument();
    expect(screen.getByText('Language')).toBeInTheDocument();
    expect(screen.getByText('Theme')).toBeInTheDocument();
    expect(screen.getByText('High contrast')).toBeInTheDocument();
    expect(screen.getByText('Two-Factor Authentication')).toBeInTheDocument();
    expect(screen.getAllByText('Switch').length).toBe(3);
    expect(screen.getByText('Disable')).toBeInTheDocument();
    expect(screen.getByText('Save profile')).toBeInTheDocument();
    expect(screen.getByText('Logout')).toBeInTheDocument();
    expect(screen.getByText('Go home')).toBeInTheDocument();
    expect(screen.getByText('Code to disable')).toBeInTheDocument();
  });

  it('shows EN subtitle when language is en', () => {
    const enContext = baseContext({
      state: { language: 'en' },
      t: (_ru, en) => en,
    });
    renderProfile(enContext);
    expect(screen.getByText('EN')).toBeInTheDocument();
  });

  it('shows ON subtitle when high contrast is true', () => {
    renderProfile(baseContext({ state: { highContrast: true } }));
    expect(screen.getByText('ON')).toBeInTheDocument();
  });

  it('shows ON subtitle when twoFactor is true', () => {
    renderProfile(baseContext({ state: { profile: { twoFactor: true } } }));
    expect(screen.getByText('ON')).toBeInTheDocument();
  });

  it('renders profile fields in subtitle', () => {
    renderProfile(baseContext());
    expect(screen.getByText('mih@example.com')).toBeInTheDocument();
  });
});