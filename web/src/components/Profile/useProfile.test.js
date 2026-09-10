import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useProfile } from './useProfile';

const mockRefreshProfile = vi.fn();
const mockUseAuth = vi.fn();
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}));

const mockGetProfile = vi.fn();
const mockUpdateProfile = vi.fn();

vi.mock('../../utils/api', () => ({
  getProfile: (...args) => mockGetProfile(...args),
  updateProfile: (...args) => mockUpdateProfile(...args),
}));

describe('useProfile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({ refreshProfile: mockRefreshProfile });
  });

  it('loads profile data', async () => {
    mockGetProfile.mockResolvedValue({
      profile: {
        full_name: 'Test User',
        age: 30,
        gender: 'male',
        height_cm: 175,
        weight_kg: 70,
        fitness_level: 'beginner',
        nutrition: 'balanced',
        allergies: [],
        contraindications: [],
        goals: ['weight_loss'],
      },
    });
    const { result } = renderHook(() => useProfile());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    expect(result.current.form.nickname).toBe('Test User');
  });

  it('sets field value', async () => {
    mockGetProfile.mockResolvedValue({ profile: { full_name: '' } });
    const { result } = renderHook(() => useProfile());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    act(() => {
      result.current.setField('nickname', 'New Name');
    });
    expect(result.current.form.nickname).toBe('New Name');
  });
});
