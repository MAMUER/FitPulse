import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import ML from './ML';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderML = (authOverrides = {}) => {
  useAuth.mockReturnValue({
    token: 'test-token',
    ...authOverrides,
  });

  return render(
    <AuthProvider>
      <ML />
    </AuthProvider>
  );
};

describe('ML', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders classification section', () => {
    renderML();

    expect(screen.getByText('Классификация состояния')).toBeInTheDocument();
    expect(screen.getByText('Анализировать')).toBeInTheDocument();
  });

  it('renders plan generation form', () => {
    renderML();

    expect(screen.getByText('Генерация плана')).toBeInTheDocument();
    expect(screen.getByLabelText('Тип тренировки')).toBeInTheDocument();
    expect(screen.getByLabelText('Длительность (недель)')).toBeInTheDocument();
  });

  it('handles classification error', async () => {
    vi.spyOn(api, 'classifyState').mockRejectedValueOnce(
      new Error('analysis failed')
    );
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderML();

    await userEvent.click(screen.getByText('Анализировать'));

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith(
        'Ошибка анализа: analysis failed'
      );
    });
  });

  it('handles plan generation error', async () => {
    vi.spyOn(api, 'generateMLPlan').mockRejectedValueOnce(
      new Error('generation failed')
    );
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith(
        'Ошибка генерации: generation failed'
      );
    });
  });
});
