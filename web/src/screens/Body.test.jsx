import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Body from './Body';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('Body', () => {
  it('loads and saves body composition', async () => {
    const loadBodyComposition = vi.fn();
    const saveBodyComposition = vi.fn().mockResolvedValue({});
    const go = vi.fn();
    useApp.mockReturnValue({
      state: {
        height: 175,
        weight: 75,
        bodyComposition: [
          { recorded_at: '2026-01-01T00:00:00Z', weight_kg: 75, bmi: 24.5 },
        ],
      },
      t: (ru, _en) => ru,
      go,
      loadBodyComposition,
      saveBodyComposition,
    });

    render(<Body />);
    expect(loadBodyComposition).toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('Рост, см'), {
      target: { value: '180' },
    });
    fireEvent.change(screen.getByLabelText('Вес, кг'), {
      target: { value: '80' },
    });
    fireEvent.change(screen.getByLabelText('Жир, %'), {
      target: { value: '18' },
    });
    // Muscle mass is tracked in kilograms, not percent.
    fireEvent.change(screen.getByLabelText('Мышцы, кг'), {
      target: { value: '35' },
    });
    fireEvent.change(screen.getByLabelText('Вода, %'), {
      target: { value: '50' },
    });
    fireEvent.click(screen.getByText('Сохранить'));
    await waitFor(() => expect(saveBodyComposition).toHaveBeenCalled());

    const payload = saveBodyComposition.mock.calls[0][0];
    expect(payload).toMatchObject({
      height_cm: 180,
      weight_kg: 80,
      body_fat_percent: 18,
      muscle_mass_kg: 35,
      water_percent: 50,
    });
  });

  it('renders history records provided by weightHistory', () => {
    const loadBodyComposition = vi.fn();
    const saveBodyComposition = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: {
        height: 170,
        weight: 70,
        bodyComposition: [{ weight_kg: 70, bmi: 24.2 }],
        weightHistory: [['01.02.2026', '70 кг · ИМТ 24.2']],
      },
      t: (ru, _en) => ru,
      go,
      loadBodyComposition,
      saveBodyComposition,
    });
    render(<Body />);
    expect(screen.getByText('70 кг · ИМТ 24.2')).toBeInTheDocument();
  });

  it('renders with default height and weight when missing', () => {
    const loadBodyComposition = vi.fn();
    const saveBodyComposition = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: { bodyComposition: null },
      t: (ru, _en) => ru,
      go,
      loadBodyComposition,
      saveBodyComposition,
    });
    render(<Body />);
    expect(loadBodyComposition).toHaveBeenCalled();
    expect(screen.getByText('Аналитика тела')).toBeInTheDocument();
  });

  it('falls back to the latest values when optional fields are empty', async () => {
    const loadBodyComposition = vi.fn();
    const saveBodyComposition = vi.fn().mockResolvedValue({});
    const go = vi.fn();
    useApp.mockReturnValue({
      state: { height: 175, weight: 75, bodyComposition: [] },
      t: (ru, _en) => ru,
      go,
      loadBodyComposition,
      saveBodyComposition,
    });
    render(<Body />);
    fireEvent.click(screen.getByText('Сохранить'));
    await waitFor(() => expect(saveBodyComposition).toHaveBeenCalled());

    const arg = saveBodyComposition.mock.calls[0][0];
    // Empty inputs fall back to the latest record, or to the built-in defaults
    // when there is no history at all.
    expect(arg.height_cm).toBe(178);
    expect(arg.weight_kg).toBe(74.2);
    expect(arg.body_fat_percent).toBe(18.4);
    expect(arg.muscle_mass_kg).toBe(34.8);
    expect(arg.water_percent).toBe(57.2);
  });

  it('sorts history by recorded_at and opens each metric detail', () => {
    const openMetric = vi.fn();
    useApp.mockReturnValue({
      state: {
        height: 180,
        weight: 80,
        bodyComposition: [
          { recorded_at: '2026-01-01T00:00:00Z', weight_kg: 78 },
          { recorded_at: '2026-03-01T00:00:00Z', weight_kg: 81 },
        ],
      },
      t: (ru, _en) => ru,
      go: vi.fn(),
      openMetric,
      loadBodyComposition: vi.fn(),
      saveBodyComposition: vi.fn(),
    });

    render(<Body />);

    // Each metric card opens its detail view.
    fireEvent.click(screen.getByRole('button', { name: /Вес/ }));
    expect(openMetric).toHaveBeenLastCalledWith('weight');
    fireEvent.click(screen.getByRole('button', { name: /Жир/ }));
    expect(openMetric).toHaveBeenLastCalledWith('fat');
    fireEvent.click(screen.getByRole('button', { name: /Мышцы/ }));
    expect(openMetric).toHaveBeenLastCalledWith('muscle');
    fireEvent.click(screen.getByRole('button', { name: /Белок/ }));
    expect(openMetric).toHaveBeenLastCalledWith('protein');
    fireEvent.click(screen.getByRole('button', { name: /график/ }));
    expect(openMetric).toHaveBeenLastCalledWith('bmi');
  });

  it('opens the body-metrics detail for a vital sign', () => {
    const openMetric = vi.fn();
    useApp.mockReturnValue({
      state: { height: 180, weight: 80, bodyComposition: [] },
      t: (ru, _en) => ru,
      go: vi.fn(),
      openMetric,
      loadBodyComposition: vi.fn(),
      saveBodyComposition: vi.fn(),
    });

    render(<Body />);
    fireEvent.click(screen.getByRole('button', { name: /Пульс/ }));

    expect(openMetric).toHaveBeenCalledWith('pulse');
  });
});
