import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import Diet from './Diet';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const mockGetRandomValues = () => {
  const arr = new Uint32Array(1);
  arr[0] = 1;
  return arr;
};

const renderDiet = () => {
  useAuth.mockReturnValue({
    token: 'test-token',
  });

  return render(
    <AuthProvider>
      <Diet />
    </AuthProvider>
  );
};

describe('Diet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(window, 'crypto', {
      value: { getRandomValues: mockGetRandomValues },
      writable: true,
      configurable: true,
    });
  });

  const profile = (overrides = {}) => ({
    profile: {
      height_cm: 170,
      weight_kg: 70,
      age: 30,
      gender: 'male',
      fitness_level: 'beginner',
      goals: [],
      allergies: [],
      contraindications: [],
      ...overrides,
    },
  });

  it('shows loading state initially', () => {
    vi.spyOn(api, 'getProfile').mockImplementation(() => new Promise(() => {}));
    renderDiet();

    expect(screen.getByText('Загрузка...')).toBeInTheDocument();
  });

  it('displays profile error state', async () => {
    vi.spyOn(api, 'getProfile').mockRejectedValueOnce(new Error('load failed'));
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('Ошибка загрузки профиля')).toBeInTheDocument();
    });
  });

  it('loads profile from flat response without profile wrapper', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce({
      height_cm: 175,
      weight_kg: 70,
      age: 25,
      gender: 'male',
      fitness_level: 'beginner',
      goals: ['general_health'],
      allergies: ['nuts'],
      contraindications: ['broccoli'],
    });
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText(/ккал в день/)).toBeInTheDocument();
    });

    const allergiesInput = screen.getByLabelText('Аллергии (через запятую)');
    const dislikesInput = screen.getByLabelText(
      'Нелюбимые продукты (через запятую)'
    );
    expect(allergiesInput).toHaveValue('nuts');
    expect(dislikesInput).toHaveValue('broccoli');
  });

  it('loads profile and displays nutrition summary', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({
        height_cm: 175,
        weight_kg: 70,
        age: 25,
        gender: 'female',
        fitness_level: 'intermediate',
        goals: ['general_health'],
      })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText(/ккал в день/)).toBeInTheDocument();
    });
  });

  it.each([
    ['weight_loss', 'Похудение'],
    ['muscle_gain', 'Высокобелковое'],
    ['general_health', 'Сбалансированное'],
  ])('applies %s template based on goal', async (goal, expectedTemplate) => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({ goals: [goal] })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText(expectedTemplate)).toBeInTheDocument();
    });
  });

  it('switches template buttons', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('Сбалансированное')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Похудение'));
    expect(screen.getByText('Похудение')).toHaveClass('selected');
  });

  it('changes meal count', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const mealCountSelect = screen.getByLabelText('Количество приёмов пищи');
    await userEvent.selectOptions(mealCountSelect, '3');

    expect(mealCountSelect).toHaveValue('3');
  });

  it('changes first meal time', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const timeInput = screen.getByLabelText('Время первого приёма');
    await userEvent.clear(timeInput);
    await userEvent.type(timeInput, '09:00');

    expect(timeInput).toHaveValue('09:00');
  });

  it('sets allergies and dislikes from profile', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({
        allergies: ['nuts', 'dairy'],
        contraindications: ['broccoli', 'fish'],
      })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const allergiesInput = screen.getByLabelText('Аллергии (через запятую)');
    const dislikesInput = screen.getByLabelText(
      'Нелюбимые продукты (через запятую)'
    );

    expect(allergiesInput).toHaveValue('nuts, dairy');
    expect(dislikesInput).toHaveValue('broccoli, fish');
  });

  it('uses default values for BMI calculation when profile fields are missing', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({
        height_cm: undefined,
        weight_kg: undefined,
        age: undefined,
        gender: undefined,
        fitness_level: undefined,
        goals: [],
      })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText(/ккал в день/)).toBeInTheDocument();
    });
  });

  it('changes meal count to 3', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const mealCountSelect = screen.getByLabelText('Количество приёмов пищи');
    fireEvent.change(mealCountSelect, { target: { value: '3' } });

    await waitFor(() => {
      expect(document.querySelectorAll('.meal-card')).toHaveLength(3);
    });
  });

  it('updates allergies input', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const allergiesInput = screen.getByLabelText('Аллергии (через запятую)');
    await userEvent.type(allergiesInput, 'орехи');

    expect(allergiesInput).toHaveValue('орехи');
  });

  it('filters meals by allergy', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({ allergies: ['орехи'] })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const meals = screen.getAllByText(/ккал/);
    expect(meals.length).toBeGreaterThan(0);
  });

  it('filters meals by dislike', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({ contraindications: ['лосось'] })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const meals = screen.getAllByText(/ккал/);
    expect(meals.length).toBeGreaterThan(0);
  });

  it('updates dislikes input', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const dislikesInput = screen.getByLabelText(
      'Нелюбимые продукты (через запятую)'
    );
    await userEvent.type(dislikesInput, 'брокколи');

    expect(dislikesInput).toHaveValue('брокколи');
  });

  it('displays empty state when all meals are filtered out', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({
        allergies: ['каша', 'йогурт', 'курица', 'протеин', 'лосось'],
      })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('Нет подходящих блюд')).toBeInTheDocument();
    });
  });

  it('displays meal cards with times', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(document.querySelectorAll('.meal-card').length).toBeGreaterThan(0);
    });
  });

  it('displays nutrition totals', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('Итого за день')).toBeInTheDocument();
    });

    expect(screen.getByText('Калории')).toBeInTheDocument();
    expect(screen.getAllByText('Белки').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Жиры').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Углеводы').length).toBeGreaterThanOrEqual(1);
  });

  it('renders fitness label for beginner', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({ fitness_level: 'beginner' })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText(/Начинающий/)).toBeInTheDocument();
    });
  });

  it('renders fitness label for intermediate', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({ fitness_level: 'intermediate' })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText(/Средний/)).toBeInTheDocument();
    });
  });

  it('renders fitness label for advanced', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({ fitness_level: 'advanced' })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText(/Продвинутый/)).toBeInTheDocument();
    });
  });

  it('defaults to balanced template when no goals', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile({ goals: [] }));
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('Сбалансированное')).toBeInTheDocument();
    });
  });

  it('uses default multiplier for unknown fitness level', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({ fitness_level: 'unknown' })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText(/ккал в день/)).toBeInTheDocument();
    });
  });

  it('changes meal count to 3 using fireEvent', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const mealCountSelect = screen.getByLabelText('Количество приёмов пищи');
    fireEvent.change(mealCountSelect, { target: { value: '3' } });

    await waitFor(() => {
      expect(document.querySelectorAll('.meal-card')).toHaveLength(3);
    });
  });

  it('updates first meal time', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const timeInput = screen.getByLabelText('Время первого приёма');
    await userEvent.clear(timeInput);
    await userEvent.type(timeInput, '09:00');

    expect(timeInput).toHaveValue('09:00');
  });

  it('falls back to balanced template for unknown template key', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(
      profile({ goals: ['unknown_goal'] })
    );
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('Сбалансированное')).toBeInTheDocument();
    });
  });

  it('calculates meal times with spacing when mealCount > 1', async () => {
    vi.spyOn(api, 'getProfile').mockResolvedValueOnce(profile());
    renderDiet();

    await waitFor(() => {
      expect(screen.getByText('План питания на сегодня')).toBeInTheDocument();
    });

    const mealCountSelect = screen.getByLabelText('Количество приёмов пищи');
    await userEvent.selectOptions(mealCountSelect, '3');

    const timeInput = screen.getByLabelText('Время первого приёма');
    await userEvent.clear(timeInput);
    await userEvent.type(timeInput, '08:00');

    await waitFor(() => {
      const mealCards = document.querySelectorAll('.meal-card');
      expect(mealCards).toHaveLength(3);
    });

    const times = Array.from(document.querySelectorAll('.meal-time')).map(
      (el) => el.textContent
    );
    expect(times).toEqual(['08:00', '15:00', '22:00']);
  });
});
