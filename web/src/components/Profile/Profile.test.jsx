import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthProvider } from '../../contexts/AuthContext';
import Profile from './Profile';

const mockUseProfile = vi.fn();
vi.mock('./useProfile', () => ({
  useProfile: () => mockUseProfile(),
}));

describe('Profile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    mockUseProfile.mockReturnValue({
      loading: true,
      saving: false,
      errors: {},
      toast: '',
      form: {},
      bmi: null,
      setField: vi.fn(),
      handleSubmit: vi.fn(),
    });
    render(
      <AuthProvider>
        <Profile />
      </AuthProvider>
    );
    expect(screen.getByText('Загрузка профиля...')).toBeDefined();
  });

  it('renders profile form when loaded', () => {
    mockUseProfile.mockReturnValue({
      loading: false,
      saving: false,
      errors: {},
      toast: '',
      form: { nickname: '', age: '', gender: '', height: '', weight: '', fitness: '', nutrition: '', allergies: '', contraindications: '', goal: '' },
      bmi: null,
      setField: vi.fn(),
      handleSubmit: vi.fn(),
    });
    render(
      <AuthProvider>
        <Profile />
      </AuthProvider>
    );
    expect(screen.getByLabelText('Никнейм *')).toBeDefined();
    expect(screen.getByLabelText('Возраст')).toBeDefined();
  });

  it('opens change password modal', async () => {
    const user = userEvent.setup();
    mockUseProfile.mockReturnValue({
      loading: false,
      saving: false,
      errors: {},
      toast: '',
      form: {},
      bmi: null,
      setField: vi.fn(),
      handleSubmit: vi.fn(),
    });
    render(
      <AuthProvider>
        <Profile />
      </AuthProvider>
    );
    await user.click(screen.getByText('Сменить пароль'));
    expect(screen.getByLabelText('Текущий пароль')).toBeDefined();
  });

  it('opens change email modal', async () => {
    const user = userEvent.setup();
    mockUseProfile.mockReturnValue({
      loading: false,
      saving: false,
      errors: {},
      toast: '',
      form: {},
      bmi: null,
      setField: vi.fn(),
      handleSubmit: vi.fn(),
    });
    render(
      <AuthProvider>
        <Profile />
      </AuthProvider>
    );
    await user.click(screen.getByText('Сменить почту'));
    expect(screen.getByLabelText('Новый email')).toBeDefined();
  });

  it('opens delete profile modal', async () => {
    const user = userEvent.setup();
    mockUseProfile.mockReturnValue({
      loading: false,
      saving: false,
      errors: {},
      toast: '',
      form: {},
      bmi: null,
      setField: vi.fn(),
      handleSubmit: vi.fn(),
    });
    render(
      <AuthProvider>
        <Profile />
      </AuthProvider>
    );
    await user.click(screen.getByText('Удалить аккаунт'));
    expect(screen.getByText('Удаление аккаунта')).toBeDefined();
  });
});
