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
    vi.clearAllMocks();
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
});
