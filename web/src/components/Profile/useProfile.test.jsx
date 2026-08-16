import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useProfile } from './useProfile';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

vi.mock('../../utils/api');
vi.mock('../../utils/validators');

describe('useProfile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads profile successfully', async () => {
    const mockProfile = {
      full_name: 'Test User',
      age: 25,
      gender: 'male',
      height_cm: 175,
      weight_kg: 70,
      fitness_level: 'intermediate',
      nutrition: 'balanced',
      goals: ['weight_loss'],
      allergies: ['peanuts'],
      contraindications: ['asthma'],
    };

    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile: vi.fn() });

    const { getProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: mockProfile });

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.form.nickname).toBe('Test User');
    expect(result.current.form.age).toBe(25);
    expect(result.current.form.height).toBe(175);
    expect(result.current.form.weight).toBe(70);
    expect(result.current.form.goal).toBe('weight_loss');
    expect(result.current.form.allergies).toBe('peanuts');
    expect(result.current.form.contraindications).toBe('asthma');
  });

  it('handles load profile error', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile: vi.fn() });

    const { getProfile } = await import('../../utils/api');
    getProfile.mockRejectedValue(new Error('Network error'));

    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      'Failed to load profile:',
      expect.any(Error)
    );
    consoleSpy.mockRestore();
  });

  it('sets field value and clears error', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile: vi.fn() });

    const { getProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: {} });

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setField('nickname', 'New Name');
      result.current.setField('age', '30');
    });

    expect(result.current.form.nickname).toBe('New Name');
    expect(result.current.form.age).toBe('30');
    expect(result.current.errors.nickname).toBe('');
    expect(result.current.errors.age).toBe('');
  });

  it('shows validation errors on submit', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile: vi.fn() });

    const { getProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: {} });

    const { validateNickname, validateAge, validateHeight, validateWeight } =
      await import('../../utils/validators');
    validateNickname.mockReturnValue('Никнейм обязателен');
    validateAge.mockReturnValue('');
    validateHeight.mockReturnValue('');
    validateWeight.mockReturnValue('');

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      const mockEvent = { preventDefault: vi.fn() };
      await result.current.handleSubmit(mockEvent);
    });

    expect(result.current.errors.nickname).toBe('Никнейм обязателен');
    expect(result.current.toast).toBe('Исправьте ошибки в полях');
  });

  it('submits profile update successfully', async () => {
    const refreshProfile = vi.fn();
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile });

    const { getProfile, updateProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: {} });
    updateProfile.mockResolvedValue({});

    const { validateNickname, validateAge, validateHeight, validateWeight } =
      await import('../../utils/validators');
    validateNickname.mockReturnValue('');
    validateAge.mockReturnValue('');
    validateHeight.mockReturnValue('');
    validateWeight.mockReturnValue('');

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setField('nickname', 'Test User');
      result.current.setField('height', '175');
      result.current.setField('weight', '70');
    });

    await act(async () => {
      const mockEvent = { preventDefault: vi.fn() };
      await result.current.handleSubmit(mockEvent);
    });

    await waitFor(() => {
      expect(result.current.saving).toBe(false);
    });

    expect(updateProfile).toHaveBeenCalledWith({
      full_name: 'Test User',
      age: null,
      gender: null,
      height_cm: 175,
      weight_kg: 70,
      fitness_level: null,
      nutrition: null,
      goals: [],
      allergies: [],
      contraindications: [],
    });
    expect(refreshProfile).toHaveBeenCalled();
    expect(result.current.toast).toBe('Профиль сохранён');
  });

  it('handles submit error', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile: vi.fn() });

    const { getProfile, updateProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: {} });
    updateProfile.mockRejectedValue(new Error('Save failed'));

    const { validateNickname, validateAge, validateHeight, validateWeight } =
      await import('../../utils/validators');
    validateNickname.mockReturnValue('');
    validateAge.mockReturnValue('');
    validateHeight.mockReturnValue('');
    validateWeight.mockReturnValue('');

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setField('nickname', 'Test User');
    });

    await act(async () => {
      const mockEvent = { preventDefault: vi.fn() };
      await result.current.handleSubmit(mockEvent);
    });

    await waitFor(() => {
      expect(result.current.saving).toBe(false);
    });

    expect(result.current.toast).toBe('Ошибка: Save failed');
  });

  it('calls updateProfile with correct data on submit', async () => {
    const refreshProfile = vi.fn();
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile });

    const { getProfile, updateProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: {} });
    updateProfile.mockResolvedValue({});

    const { validateNickname, validateAge, validateHeight, validateWeight } =
      await import('../../utils/validators');
    validateNickname.mockReturnValue('');
    validateAge.mockReturnValue('');
    validateHeight.mockReturnValue('');
    validateWeight.mockReturnValue('');

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setField('nickname', 'New Name');
      result.current.setField('height', '180');
      result.current.setField('weight', '75');
    });

    await act(async () => {
      const mockEvent = { preventDefault: vi.fn() };
      await result.current.handleSubmit(mockEvent);
    });

    expect(updateProfile).toHaveBeenCalledWith(
      expect.objectContaining({
        full_name: 'New Name',
        height_cm: 180,
        weight_kg: 75,
      })
    );
    expect(result.current.toast).toBe('Профиль сохранён');
  });

  it('displays error toast when updateProfile fails', async () => {
    const refreshProfile = vi.fn();
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile });

    const { getProfile, updateProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: {} });
    updateProfile.mockRejectedValue(new Error('Server error'));

    const { validateNickname, validateAge, validateHeight, validateWeight } =
      await import('../../utils/validators');
    validateNickname.mockReturnValue('');
    validateAge.mockReturnValue('');
    validateHeight.mockReturnValue('');
    validateWeight.mockReturnValue('');

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setField('nickname', 'Test User');
    });

    await act(async () => {
      const mockEvent = { preventDefault: vi.fn() };
      await result.current.handleSubmit(mockEvent);
    });

    await waitFor(() => {
      expect(result.current.toast).toBe('Ошибка: Server error');
    });
  });

  it('loads profile from flat response without profile wrapper', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile: vi.fn() });

    const { getProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({
      full_name: 'Flat User',
      age: 25,
      gender: 'male',
      height_cm: 175,
      weight_kg: 70,
      fitness_level: 'intermediate',
      nutrition: 'balanced',
      goals: ['weight_loss'],
      allergies: ['peanuts'],
      contraindications: ['asthma'],
    });

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.form.nickname).toBe('Flat User');
    expect(result.current.form.age).toBe(25);
    expect(result.current.form.height).toBe(175);
    expect(result.current.form.weight).toBe(70);
    expect(result.current.form.goal).toBe('weight_loss');
    expect(result.current.form.allergies).toBe('peanuts');
    expect(result.current.form.contraindications).toBe('asthma');
  });

  it('shows height validation error on submit', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile: vi.fn() });

    const { getProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: {} });

    const { validateNickname, validateAge, validateHeight, validateWeight } =
      await import('../../utils/validators');
    validateNickname.mockReturnValue('');
    validateAge.mockReturnValue('');
    validateHeight.mockReturnValue('Некорректный рост');
    validateWeight.mockReturnValue('');

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setField('nickname', 'Test User');
      result.current.setField('height', '50');
    });

    await act(async () => {
      const mockEvent = { preventDefault: vi.fn() };
      await result.current.handleSubmit(mockEvent);
    });

    expect(result.current.errors.height).toBe('Некорректный рост');
  });

  it('shows weight validation error on submit', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    useAuth.mockReturnValue({ refreshProfile: vi.fn() });

    const { getProfile } = await import('../../utils/api');
    getProfile.mockResolvedValue({ profile: {} });

    const { validateNickname, validateAge, validateHeight, validateWeight } =
      await import('../../utils/validators');
    validateNickname.mockReturnValue('');
    validateAge.mockReturnValue('');
    validateHeight.mockReturnValue('');
    validateWeight.mockReturnValue('Некорректный вес');

    const { result } = renderHook(() => useProfile());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setField('nickname', 'Test User');
      result.current.setField('weight', '30');
    });

    await act(async () => {
      const mockEvent = { preventDefault: vi.fn() };
      await result.current.handleSubmit(mockEvent);
    });

    expect(result.current.errors.weight).toBe('Некорректный вес');
  });
});
