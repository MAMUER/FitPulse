import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import Health from './Health';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderHealth = () => {
  useAuth.mockReturnValue({
    token: 'test-token',
  });

  return render(
    <AuthProvider>
      <Health />
    </AuthProvider>
  );
};

describe('Health', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('shows loading state initially', () => {
    vi.spyOn(api, 'listHealthConditions').mockImplementation(
      () => new Promise(() => {})
    );
    renderHealth();

    expect(screen.getByText('Загрузка данных здоровья...')).toBeInTheDocument();
  });

  it('displays empty state for conditions', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Нет добавленных состояний')).toBeInTheDocument();
    });
  });

  it('displays health conditions', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([
      {
        condition_id: '1',
        condition_name: 'Аллергия на пыльцу',
        condition_type: 'allergy',
        severity: 'mild',
        notes: 'Летний период',
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Аллергия на пыльцу')).toBeInTheDocument();
    });
  });

  it('displays body composition records', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([
      {
        composition_id: '1',
        weight_kg: 70,
        height_cm: 175,
        body_fat_percentage: 15,
        muscle_mass_percentage: 45,
        recorded_at: '2024-01-01',
      },
    ]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Вес: 70 кг')).toBeInTheDocument();
    });
  });

  it('displays empty state for body composition', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });
  });

  it('displays menstrual cycles', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([
      {
        menstrual_cycle_id: '1',
        cycle_start_date: '2024-01-01',
        cycle_end_date: '2024-01-28',
        flow_intensity: 'medium',
      },
    ]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText(/Начало: 2024-01-01/)).toBeInTheDocument();
    });
  });

  it('displays empty state for menstrual cycles', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });
  });

  it('handles load error gracefully', async () => {
    vi.spyOn(api, 'listHealthConditions').mockRejectedValueOnce(
      new Error('load failed')
    );
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Нет добавленных состояний')).toBeInTheDocument();
    });
  });

  it('logs error when Promise.allSettled throws', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);

    const consoleErrorSpy = vi
      .spyOn(console, 'error')
      .mockImplementation(() => {});
    const originalAllSettled = Promise.allSettled;
    Promise.allSettled = vi.fn(() => {
      throw new Error('settled failed');
    });

    renderHealth();

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Failed to load health data:',
        expect.any(Error)
      );
    });

    Promise.allSettled = originalAllSettled;
    consoleErrorSpy.mockRestore();
  });

  it('deletes condition successfully', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([
      {
        condition_id: '1',
        condition_name: 'Аллергия на пыльцу',
        condition_type: 'allergy',
        severity: 'mild',
        notes: 'Летний период',
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'deleteHealthCondition').mockResolvedValueOnce({});
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Аллергия на пыльцу')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Удалить'));

    expect(api.deleteHealthCondition).toHaveBeenCalledWith('1');
  });

  it('cancels delete when confirm is false', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([
      {
        condition_id: '1',
        condition_name: 'Аллергия на пыльцу',
        condition_type: 'allergy',
        severity: 'mild',
        notes: 'Летний период',
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Аллергия на пыльцу')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Удалить'));

    expect(api.deleteHealthCondition).not.toHaveBeenCalled();
  });

  it('adds condition successfully', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'upsertHealthCondition').mockResolvedValueOnce({});
    vi.spyOn(window, 'prompt')
      .mockReturnValueOnce('Новая аллергия')
      .mockReturnValueOnce('allergy')
      .mockReturnValueOnce('mild')
      .mockReturnValueOnce('Заметки');
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Нет добавленных состояний')).toBeInTheDocument();
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[0]);

    expect(api.upsertHealthCondition).toHaveBeenCalledWith(
      expect.objectContaining({
        condition_name: 'Новая аллергия',
        condition_type: 'allergy',
      })
    );
  });

  it('cancels add condition when name is empty', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(window, 'prompt').mockReturnValueOnce(null);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Нет добавленных состояний')).toBeInTheDocument();
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[0]);

    expect(api.upsertHealthCondition).not.toHaveBeenCalled();
  });

  it('adds body composition successfully', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createBodyComposition').mockResolvedValueOnce({});
    vi.spyOn(window, 'prompt')
      .mockReturnValueOnce('70')
      .mockReturnValueOnce('175')
      .mockReturnValueOnce('15')
      .mockReturnValueOnce('45');
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[1]);

    expect(api.createBodyComposition).toHaveBeenCalledWith(
      expect.objectContaining({
        weight_kg: 70,
        height_cm: 175,
      })
    );
  });

  it('cancels add body composition when weight is empty', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(window, 'prompt').mockReturnValueOnce(null);
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[1]);

    expect(api.createBodyComposition).not.toHaveBeenCalled();
  });

  it('adds menstrual cycle successfully', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createMenstrualCycle').mockResolvedValueOnce({});
    vi.spyOn(window, 'prompt')
      .mockReturnValueOnce('2024-01-01')
      .mockReturnValueOnce('2024-01-28')
      .mockReturnValueOnce('medium')
      .mockReturnValueOnce('headache')
      .mockReturnValueOnce('happy');
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[2]);

    expect(api.createMenstrualCycle).toHaveBeenCalledWith(
      expect.objectContaining({
        cycle_start_date: '2024-01-01',
      })
    );
  });

  it('cancels add menstrual cycle when date is empty', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(window, 'prompt').mockReturnValueOnce(null);
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[2]);

    expect(api.createMenstrualCycle).not.toHaveBeenCalled();
  });

  it('deletes condition with error', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([
      {
        condition_id: '1',
        condition_name: 'Аллергия на пыльцу',
        condition_type: 'allergy',
        severity: 'mild',
        notes: 'Летний период',
        is_active: true,
      },
    ]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'deleteHealthCondition').mockRejectedValueOnce(
      new Error('delete failed')
    );
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Аллергия на пыльцу')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Удалить'));

    expect(api.deleteHealthCondition).toHaveBeenCalledWith('1');
  });

  it('deletes menstrual cycle successfully', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([
      {
        menstrual_cycle_id: '1',
        cycle_start_date: '2024-01-01',
        cycle_end_date: '2024-01-28',
        flow_intensity: 'medium',
      },
    ]);
    vi.spyOn(api, 'deleteMenstrualCycle').mockResolvedValueOnce({});
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText(/Начало: 2024-01-01/)).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Удалить'));

    expect(api.deleteMenstrualCycle).toHaveBeenCalledWith('1');
  });

  it('deletes menstrual cycle with error', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([
      {
        menstrual_cycle_id: '1',
        cycle_start_date: '2024-01-01',
        cycle_end_date: '2024-01-28',
        flow_intensity: 'medium',
      },
    ]);
    vi.spyOn(api, 'deleteMenstrualCycle').mockRejectedValueOnce(
      new Error('delete failed')
    );
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText(/Начало: 2024-01-01/)).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Удалить'));

    expect(api.deleteMenstrualCycle).toHaveBeenCalledWith('1');
  });

  it('handles add condition error', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'upsertHealthCondition').mockRejectedValueOnce(
      new Error('add failed')
    );
    vi.spyOn(window, 'prompt')
      .mockReturnValueOnce('Новая аллергия')
      .mockReturnValueOnce('allergy')
      .mockReturnValueOnce('mild')
      .mockReturnValueOnce('Заметки');
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Нет добавленных состояний')).toBeInTheDocument();
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[0]);

    expect(screen.getByText('Ошибка: add failed')).toBeInTheDocument();
  });

  it('handles add body composition error', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createBodyComposition').mockRejectedValueOnce(
      new Error('add failed')
    );
    vi.spyOn(window, 'prompt')
      .mockReturnValueOnce('70')
      .mockReturnValueOnce('175')
      .mockReturnValueOnce('15')
      .mockReturnValueOnce('45');
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[1]);

    expect(screen.getByText('Ошибка: add failed')).toBeInTheDocument();
  });

  it('handles add menstrual cycle error', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createMenstrualCycle').mockRejectedValueOnce(
      new Error('add failed')
    );
    vi.spyOn(window, 'prompt')
      .mockReturnValueOnce('2024-01-01')
      .mockReturnValueOnce('2024-01-28')
      .mockReturnValueOnce('medium')
      .mockReturnValueOnce('headache')
      .mockReturnValueOnce('happy');
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[2]);

    expect(screen.getByText('Ошибка: add failed')).toBeInTheDocument();
  });

  it('displays condition with missing fields', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([
      {
        condition_id: '1',
        condition_name: '',
        condition_type: '',
        severity: '',
        notes: '',
        is_active: false,
      },
    ]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Без названия')).toBeInTheDocument();
    });

    expect(screen.getByText('Другое')).toBeInTheDocument();
  });

  it('displays body composition with missing date', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([
      {
        composition_id: '1',
        weight_kg: 70,
        height_cm: 175,
        recorded_at: '',
      },
    ]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Вес: 70 кг')).toBeInTheDocument();
    });
  });

  it('displays menstrual cycle with missing fields', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([
      {
        menstrual_cycle_id: '1',
        cycle_start_date: '',
        cycle_end_date: '',
        flow_intensity: '',
        symptoms: [],
        moods: [],
        notes: '',
      },
    ]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Цикл')).toBeInTheDocument();
    });

    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('adds body composition with empty optional fields', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createBodyComposition').mockResolvedValueOnce({});
    vi.spyOn(window, 'prompt')
      .mockReturnValueOnce('70')
      .mockReturnValueOnce('175')
      .mockReturnValueOnce('')
      .mockReturnValueOnce('');
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[1]);

    expect(api.createBodyComposition).toHaveBeenCalledWith(
      expect.objectContaining({
        weight_kg: 70,
        height_cm: 175,
        body_fat_percentage: null,
        muscle_mass_percentage: null,
      })
    );
  });

  it('adds menstrual cycle with empty optional fields', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    vi.spyOn(api, 'createMenstrualCycle').mockResolvedValueOnce({});
    vi.spyOn(window, 'prompt')
      .mockReturnValueOnce('2024-01-01')
      .mockReturnValueOnce('')
      .mockReturnValueOnce('medium')
      .mockReturnValueOnce('')
      .mockReturnValueOnce('');
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderHealth();

    await waitFor(() => {
      expect(screen.getAllByText('Нет записей').length).toBeGreaterThanOrEqual(
        1
      );
    });

    const addButtons = screen.getAllByText('Добавить');
    await userEvent.click(addButtons[2]);

    expect(api.createMenstrualCycle).toHaveBeenCalledWith(
      expect.objectContaining({
        cycle_start_date: '2024-01-01',
        cycle_end_date: null,
        symptoms: [],
        moods: [],
      })
    );
  });

  it('renders condition with notes and inactive status', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([
      {
        condition_id: '1',
        condition_name: 'Тест',
        condition_type: 'disease',
        severity: 'high',
        notes: 'Заметки',
        is_active: false,
      },
    ]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Тест')).toBeInTheDocument();
    });

    expect(screen.getByText('Заметки')).toBeInTheDocument();
    expect(screen.getByText('Неактивно')).toBeInTheDocument();
  });

  it('renders body composition with all fields', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([
      {
        composition_id: '1',
        weight_kg: 70,
        height_cm: 175,
        body_fat_percentage: 15,
        muscle_mass_percentage: 45,
        recorded_at: '2024-01-01',
      },
    ]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText('Вес: 70 кг')).toBeInTheDocument();
    });

    expect(screen.getByText('Жир: 15%')).toBeInTheDocument();
    expect(screen.getByText('Мышцы: 45%')).toBeInTheDocument();
  });

  it('renders menstrual cycle with symptoms and moods', async () => {
    vi.spyOn(api, 'listHealthConditions').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listBodyComposition').mockResolvedValueOnce([]);
    vi.spyOn(api, 'listMenstrualCycles').mockResolvedValueOnce([
      {
        menstrual_cycle_id: '1',
        cycle_start_date: '2024-01-01',
        cycle_end_date: '2024-01-28',
        flow_intensity: 'medium',
        symptoms: ['headache', 'cramps'],
        moods: ['happy', 'sad'],
        notes: 'Test notes',
      },
    ]);
    renderHealth();

    await waitFor(() => {
      expect(screen.getByText(/Начало: 2024-01-01/)).toBeInTheDocument();
    });

    expect(screen.getByText('Симптомы: headache, cramps')).toBeInTheDocument();
    expect(screen.getByText('Настроения: happy, sad')).toBeInTheDocument();
    expect(screen.getByText('Test notes')).toBeInTheDocument();
  });
});
