import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import { getProfile, updateProfile } from '../../utils/api';
import { calculateBMI } from '../../utils/validators';
import Profile from './Profile';

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
  default: ({ onClose }) => (
    <div data-testid='password-modal'>
      Password Modal
      <button type='button' onClick={onClose}>
        Close
      </button>
    </div>
  ),
}));

vi.mock('./ChangeEmailModal', () => ({
  __esModule: true,
  default: ({ onClose }) => (
    <div data-testid='email-modal'>
      Email Modal
      <button type='button' onClick={onClose}>
        Close
      </button>
    </div>
  ),
}));

vi.mock('./DeleteProfileModal', () => ({
  __esModule: true,
  default: ({ onClose }) => (
    <div data-testid='delete-modal'>
      Delete Modal
      <button type='button' onClick={onClose}>
        Close
      </button>
    </div>
  ),
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

  const user = userEvent.setup();

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

    await user.click(screen.getByText('Сохранить'));

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

    await user.click(screen.getByText('Сохранить'));

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

    await user.click(screen.getByText(button));

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

  it('allows typing in form fields', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('Test')).toBeInTheDocument();
    });

    const nicknameInput = screen.getByLabelText('Никнейм *');
    await user.clear(nicknameInput);
    await user.type(nicknameInput, 'NewName');
    expect(nicknameInput).toHaveValue('NewName');
  });

  it('allows typing in age field', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('25')).toBeInTheDocument();
    });

    const ageInput = screen.getByLabelText('Возраст');
    await user.clear(ageInput);
    await user.type(ageInput, '30');
    expect(ageInput).toHaveValue(30);
  });

  it('allows typing in height field', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    });

    const heightInput = screen.getByLabelText('Рост, см');
    await user.clear(heightInput);
    await user.type(heightInput, '180');
    expect(heightInput).toHaveValue(180);
  });

  it('allows typing in weight field', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('70')).toBeInTheDocument();
    });

    const weightInput = screen.getByLabelText('Вес, кг');
    await user.clear(weightInput);
    await user.type(weightInput, '75');
    expect(weightInput).toHaveValue(75);
  });

  it('allows changing goal selection', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    });

    const muscleGainLabel = screen.getByText('Набор мышц');
    await user.click(muscleGainLabel);
    expect(muscleGainLabel.closest('label')).toHaveClass('selected');
  });

  it('opens and closes password modal', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByText('Сменить пароль')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Сменить пароль'));
    expect(screen.getByTestId('password-modal')).toBeInTheDocument();

    await user.click(screen.getByText('Close'));
    expect(screen.queryByTestId('password-modal')).not.toBeInTheDocument();
  });

  it('opens and closes email modal', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByText('Сменить почту')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Сменить почту'));
    expect(screen.getByTestId('email-modal')).toBeInTheDocument();

    await user.click(screen.getByText('Close'));
    expect(screen.queryByTestId('email-modal')).not.toBeInTheDocument();
  });

  it('opens and closes delete modal', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByText('Удалить аккаунт')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Удалить аккаунт'));
    expect(screen.getByTestId('delete-modal')).toBeInTheDocument();

    await user.click(screen.getByText('Close'));
    expect(screen.queryByTestId('delete-modal')).not.toBeInTheDocument();
  });

  it('allows typing in allergies field', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    });

    const allergiesInput = screen.getByPlaceholderText(
      'Например: орехи, лактоза, глютен'
    );
    await user.type(allergiesInput, 'орехи, лактоза');
    expect(allergiesInput).toHaveValue('орехи, лактоза');
  });

  it('allows typing in contraindications field', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    });

    const contraindicationsInput = screen.getByPlaceholderText(
      'Например: проблемы с коленями, астма'
    );
    await user.type(contraindicationsInput, 'проблемы с коленями');
    expect(contraindicationsInput).toHaveValue('проблемы с коленями');
  });

  it('allows changing fitness level', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    });

    const fitnessSelect = screen.getByLabelText('Уровень подготовки');
    await user.selectOptions(fitnessSelect, 'beginner');
    expect(fitnessSelect).toHaveValue('beginner');
  });

  it('allows changing nutrition type', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
      expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    });

    const nutritionSelect = screen.getByLabelText('Тип питания');
    await user.selectOptions(nutritionSelect, 'high_protein');
    expect(nutritionSelect).toHaveValue('high_protein');
  });

  it('displays toast message on save error', async () => {
    getProfile.mockResolvedValueOnce({
      profile: {
        full_name: 'Test',
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
    updateProfile.mockRejectedValueOnce(new Error('save failed'));
    renderProfile();

    await waitFor(() => {
      expect(screen.getByDisplayValue('175')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Сохранить'));

    await waitFor(() => {
      expect(screen.getByText(/Ошибка: save failed/)).toBeInTheDocument();
    });
  });
});
