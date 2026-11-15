import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
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
    activeDays: 12,
    onlineUsers: 8,
    profile: {
      name: 'Михаил',
      email: 'mih@example.com',
      twoFactor: false,
      devices: [],
    },
    friends: [],
    achievements: [],
    ...overrides.state,
  },
  t: (ru) => ru,
  go: vi.fn(),
  openLegal: vi.fn(),
  toggleLanguage: vi.fn(),
  toggleTheme: vi.fn(),
  toggleHighContrast: vi.fn(),
  load2FAStatus: vi.fn(),
  setup2FA: vi.fn(),
  disable2FA: vi.fn().mockResolvedValue({}),
  loadProfile: vi.fn(),
  changePassword: vi.fn(),
  deleteAccount: vi.fn().mockReturnValue(true),
  uploadPhoto: vi.fn(),
  editProfile: vi.fn(),
  showDevices: vi.fn(),
  showRestrictions: vi.fn(),
  nutritionSettings: vi.fn(),
  saveChatSettings: vi.fn(),
  searchFriends: vi.fn(),
  openFriendProfile: vi.fn(),
  showAchievement: vi.fn(),
  resumeSurvey: vi.fn(),
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

  it('renders the profile card, achievements, friends, and settings', () => {
    renderProfile(
      baseContext({
        state: {
          profile: { name: 'Михаил', email: 'mih@example.com', devices: [] },
          achievements: [{ name: 'Первое достижение', done: true }],
          friends: [{ id: 1, name: 'Анна', online: true, status: 'gym' }],
        },
      })
    );

    expect(screen.getByText('FITPULSE IDENTITY')).toBeInTheDocument();
    expect(screen.getByText('Михаил')).toBeInTheDocument();
    expect(screen.getByText('Первое достижение')).toBeInTheDocument();
    expect(screen.getByText('Анна')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Тема/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Язык/ })).toBeInTheDocument();
    expect(screen.getByText('Удалить аккаунт')).toBeInTheDocument();
  });

  it('loads profile and two-factor status on mount', () => {
    const loadProfile = vi.fn();
    const load2FAStatus = vi.fn();
    renderProfile(baseContext({ loadProfile, load2FAStatus }));
    expect(loadProfile).toHaveBeenCalledTimes(1);
    expect(load2FAStatus).toHaveBeenCalledTimes(1);
  });

  it('opens the password form from the settings item', async () => {
    const user = userEvent.setup();
    renderProfile(baseContext());

    await user.click(
      screen.getAllByRole('button', { name: /Сменить пароль/ })[0]
    );

    expect(screen.getByRole('dialog', { name: 'Смена пароля' })).toBeVisible();
    expect(screen.getByLabelText('Текущий пароль')).toBeInTheDocument();
  });

  it('saves chat privacy settings from the profile', async () => {
    const user = userEvent.setup();
    const saveChatSettings = vi.fn();
    renderProfile(baseContext({ saveChatSettings }));

    await user.click(screen.getByRole('button', { name: /Настройки чата/ }));
    expect(
      screen.getByRole('dialog', { name: 'Настройки чата' })
    ).toBeVisible();
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));

    expect(saveChatSettings).toHaveBeenCalledTimes(1);
    await waitFor(() => {
      expect(
        screen.queryByRole('dialog', { name: 'Настройки чата' })
      ).not.toBeInTheDocument();
    });
  });

  it('routes theme, language, contrast, and documents to their existing actions', () => {
    const context = baseContext();
    renderProfile(context);

    fireEvent.click(screen.getByRole('button', { name: /Тема/ }));
    fireEvent.click(screen.getByRole('button', { name: /Язык/ }));
    fireEvent.click(
      screen.getByRole('button', { name: /Режим высокой контрастности/ })
    );
    fireEvent.click(screen.getByRole('button', { name: 'Соглашение' }));

    expect(context.toggleTheme).toHaveBeenCalled();
    expect(context.toggleLanguage).toHaveBeenCalled();
    expect(context.toggleHighContrast).toHaveBeenCalled();
    expect(context.openLegal).toHaveBeenCalledWith('terms');
  });

  it('sets up two-factor authentication when disabled', async () => {
    const user = userEvent.setup();
    const setup2FA = vi.fn();
    renderProfile(baseContext({ setup2FA }));

    await user.click(screen.getByRole('button', { name: 'Включить' }));

    expect(setup2FA).toHaveBeenCalled();
  });

  it('disables two-factor authentication with the entered code', async () => {
    const user = userEvent.setup();
    const disable2FA = vi.fn().mockResolvedValue({});
    renderProfile(
      baseContext({
        state: { profile: { twoFactor: true, devices: [] } },
        disable2FA,
      })
    );

    await user.type(screen.getByLabelText('Код для отключения'), '123456');
    await user.click(screen.getByRole('button', { name: 'Отключить' }));

    await waitFor(() => {
      expect(disable2FA).toHaveBeenCalledWith('123456');
    });
  });

  it('confirms account deletion before navigating to login', async () => {
    const user = userEvent.setup();
    const context = baseContext();
    renderProfile(context);

    await user.click(screen.getByRole('button', { name: /Удалить аккаунт/ }));

    expect(context.deleteAccount).toHaveBeenCalled();
    expect(context.go).toHaveBeenCalledWith('login');
  });
});
