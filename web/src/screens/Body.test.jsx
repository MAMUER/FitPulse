import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Body from './Body';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

function ctx(overrides = {}) {
  return {
    state: { bodyComposition: [], ...overrides.state },
    t: (ru, _en) => ru,
    canEdit: true,
    go: vi.fn(),
    loadBodyComposition: vi.fn(),
    saveBodyComposition: vi.fn().mockResolvedValue({}),
    openMetric: vi.fn(),
    addWeight: vi.fn(),
    ...overrides,
  };
}

describe('Body', () => {
  it('loads and saves body composition', async () => {
    const context = ctx({
      state: {
        bodyComposition: [
          { recorded_at: '2026-01-01T00:00:00Z', weight_kg: 75, bmi: 24.5 },
        ],
      },
    });
    useApp.mockReturnValue(context);

    render(<Body />);
    expect(context.loadBodyComposition).toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('Рост, см'), {
      target: { value: '180' },
    });
    fireEvent.change(screen.getByLabelText('Вес, кг'), {
      target: { value: '80' },
    });
    fireEvent.change(screen.getByLabelText('Жир, %'), {
      target: { value: '18' },
    });
    fireEvent.change(screen.getByLabelText('Мышцы, кг'), {
      target: { value: '35' },
    });
    fireEvent.change(screen.getByLabelText('Вода, %'), {
      target: { value: '50' },
    });
    fireEvent.click(screen.getByText('Сохранить'));
    await waitFor(() => expect(context.saveBodyComposition).toHaveBeenCalled());

    expect(context.saveBodyComposition.mock.calls[0][0]).toMatchObject({
      height_cm: 180,
      weight_kg: 80,
      body_fat_percent: 18,
      muscle_mass_kg: 35,
      water_percent: 50,
    });
  });

  it('renders history records provided by weightHistory', () => {
    const context = ctx({
      state: {
        bodyComposition: [{ weight_kg: 70, bmi: 24.2 }],
        weightHistory: [['01.02.2026', '70 кг · ИМТ 24.2']],
      },
    });
    useApp.mockReturnValue(context);

    render(<Body />);
    expect(screen.getByText('70 кг · ИМТ 24.2')).toBeInTheDocument();
  });

  it('renders the page when there is no composition data', () => {
    const context = ctx({ state: { bodyComposition: null } });
    useApp.mockReturnValue(context);

    render(<Body />);
    expect(context.loadBodyComposition).toHaveBeenCalled();
    expect(screen.getByText('Аналитика тела')).toBeInTheDocument();
  });

  it('does not save an empty entry without weight and height', async () => {
    const context = ctx({ state: { bodyComposition: [] } });
    useApp.mockReturnValue(context);

    render(<Body />);
    fireEvent.click(screen.getByText('Сохранить'));

    // Нет веса/роста — запись не сохраняем.
    expect(context.saveBodyComposition).not.toHaveBeenCalled();
  });

  it('sorts history by recorded_at and opens metric details', () => {
    const context = ctx({
      state: {
        bodyComposition: [
          { recorded_at: '2026-01-01T00:00:00Z', weight_kg: 78 },
          { recorded_at: '2026-03-01T00:00:00Z', weight_kg: 81 },
        ],
      },
    });
    useApp.mockReturnValue(context);

    render(<Body />);

    fireEvent.click(screen.getByRole('button', { name: /Вес/ }));
    expect(context.openMetric).toHaveBeenLastCalledWith('weight');
    fireEvent.click(screen.getByRole('button', { name: /Жир/ }));
    expect(context.openMetric).toHaveBeenLastCalledWith('fat');
    fireEvent.click(screen.getByRole('button', { name: /Мышцы/ }));
    expect(context.openMetric).toHaveBeenLastCalledWith('muscle');
    fireEvent.click(screen.getByRole('button', { name: /график/ }));
    expect(context.openMetric).toHaveBeenLastCalledWith('bmi');
  });

  it('opens the body-metrics detail for a vital sign', () => {
    const context = ctx({ state: { bodyComposition: [] } });
    useApp.mockReturnValue(context);

    render(<Body />);
    fireEvent.click(screen.getByRole('button', { name: /Пульс/ }));

    expect(context.openMetric).toHaveBeenCalledWith('pulse');
  });

  it('uses the latest heightCm when the height field is left empty', async () => {
    const context = ctx({
      state: {
        bodyComposition: [
          { recorded_at: '2026-01-01T00:00:00Z', heightCm: 175 },
        ],
      },
    });
    useApp.mockReturnValue(context);

    render(<Body />);
    fireEvent.change(screen.getByLabelText('Вес, кг'), {
      target: { value: '80' },
    });
    fireEvent.click(screen.getByText('Сохранить'));
    await waitFor(() => expect(context.saveBodyComposition).toHaveBeenCalled());

    expect(context.saveBodyComposition.mock.calls[0][0]).toMatchObject({
      height_cm: 175,
      weight_kg: 80,
    });
  });

  it('does not save when the stored height is zero', async () => {
    const context = ctx({
      state: { bodyComposition: [{ height_cm: 0, weight_kg: 80 }] },
    });
    useApp.mockReturnValue(context);

    render(<Body />);
    fireEvent.click(screen.getByText('Сохранить'));

    // Без роста ИМТ не считаем — запись пропускаем.
    expect(context.saveBodyComposition).not.toHaveBeenCalled();
  });
});
