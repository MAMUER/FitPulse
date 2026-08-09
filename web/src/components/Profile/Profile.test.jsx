import { act, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Profile from './Profile';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import { getProfile, updateProfile } from '../../utils/api';
import { calculateBMI } from '../../utils/validators';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

vi.mock('../../utils/api', () => ({
  getProfile: vi.fn(),
  updateProfile: vi.fn(),
}));

vi.mock('./ChangePasswordModal', () => ({
  __esModule: true,
  default: ({ onClose }) => <div data-testid='password-modal'>Password Modal<button onClick={onClose}>Close</button></div>,
}));

vi.mock('./ChangeEmailModal', () => ({
  __esModule: true,
  default: ({ onClose }) => <div data-testid='email-modal'>Email Modal<button onClick={onClose}>Close</button></div>,
}));

vi.mock('./DeleteProfileModal', () => ({
  __esModule: true,
  default: ({ onClose }) => <div data-testid='delete-modal'>Delete Modal<button onClick={onClose}>Close</button></div>,
}));

vi.mock('./TwoFASetup', () => ({
  __esModule: true,
  default: () => <div data-testid='twofa-setup'>2FA Setup</div>,
}));

const renderProfile = (authOverrides = {}) => {
  useAuth.mockReturnValue({
    refreshProfile: vi.fn(),
    ...authOverrides,
  });

  return render(
    <AuthProvider>
      <Profile />
    </AuthProvider>
  );
};

describe('Profile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state initially', () => {
    getProfile.mockImplementation(() => new Promise(() => {}));
    renderProfile();
    expect(screen.getByText('Загрузка профиля...')).toBeInTheDocument();
  });

  it('loads and displays profile data', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test User',
        age: 25,
        gender: 'male',
        height_cm: 175,
        weight_kg: 70,
        fitness_level: 'intermediate',
        nutrition: 'balanced',
        allergies: [],
        contraindications: [],
        goals: ['weight_loss'],
      },
    });
    renderProfile();

    await waitFor(() => {
      expect(screen.getByDisplayValue('Test User')).toBeInTheDocument();
    });

    expect(screen.getByDisplayValue('25')).toBeInTheDocument();
    expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    expect(screen.getByDisplayValue('70')).toBeInTheDocument();
  });

  it('displays BMI when height and weight are set', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
        age: 25,
        gender: '',
        height_cm: 175,
        weight_kg: 70,
        fitness_level: '',
        nutrition: '',
        allergies: [],
        contraindications: [],
        goals: [],
      },
    });
    renderProfile();

    await waitFor(() => {
      expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    });

    const bmi = calculateBMI(175, 70);
    expect(screen.getByText(new RegExp(bmi.bmi))).toBeInTheDocument();
  });

  it('shows validation errors on submit', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: '',
        age: '',
        gender: '',
        height_cm: '',
        weight_kg: '',
        fitness_level: '',
        nutrition: '',
        allergies: [],
        contraindications: [],
        goals: [],
      },
    });
    renderProfile();

    await waitFor(() => {
      expect(screen.getByText('Сохранить')).toBeInTheDocument();
    });

    screen.getByText('Сохранить').click();

    await waitFor(() => {
      expect(screen.getByText('Никнейм обязателен')).toBeInTheDocument();
    });
  });

  it('submits profile successfully', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test User',
        age: 25,
        gender: 'male',
        height_cm: 175,
        weight_kg: 70,
        fitness_level: 'intermediate',
        nutrition: 'balanced',
        allergies: [],
        contraindications: [],
        goals: ['weight_loss'],
      },
    });
    updateProfile.mockResolvedValueOnce({});
    renderProfile();

    await waitFor(() => {
      expect(screen.getByDisplayValue('Test User')).toBeInTheDocument();
    });

    screen.getByText('Сохранить').click();

    await waitFor(() => {
      expect(screen.getByText('Профиль сохранён')).toBeInTheDocument();
    });
  });

  test.each([
    { button: 'Сменить пароль', modal: 'password-modal' },
    { button: 'Сменить почту', modal: 'email-modal' },
    { button: 'Удалить аккаунт', modal: 'delete-modal' },
  ])('opens $button', async ({ button, modal }) => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
        age: '',
        gender: '',
        height_cm: '',
        weight_kg: '',
        fitness_level: '',
        nutrition: '',
        allergies: [],
        contraindications: [],
        goals: [],
      },
    });
    renderProfile();

    await waitFor(() => {
      expect(screen.getByText(button)).toBeInTheDocument();
    });

    await act(async () => {
      screen.getByText(button).click();
    });

    expect(screen.getByTestId(modal)).toBeInTheDocument();
  });

  it('renders TwoFASetup component', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
        age: '',
        gender: '',
        height_cm: '',
        weight_kg: '',
        fitness_level: '',
        nutrition: '',
        allergies: [],
        contraindications: [],
        goals: [],
      },
    });
    renderProfile();

    await waitFor(() => {
      expect(screen.getByTestId('twofa-setup')).toBeInTheDocument();
    });
  });
});
