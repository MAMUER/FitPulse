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
});
