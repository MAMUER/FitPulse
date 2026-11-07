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
    fireEvent.change(screen.getByLabelText('Мышцы, %'), {
      target: { value: '35' },
    });
    fireEvent.change(screen.getByLabelText('Вода, %'), {
      target: { value: '50' },
    });
    fireEvent.click(screen.getByText('Сохранить'));
    await waitFor(() => expect(saveBodyComposition).toHaveBeenCalled());
    fireEvent.click(screen.getByText('На главную'));
    expect(go).toHaveBeenCalledWith('home');
  });

  it('renders history record without recorded_at or id', () => {
    const loadBodyComposition = vi.fn();
    const saveBodyComposition = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: {
        height: 170,
        weight: 70,
        bodyComposition: [{ weight_kg: 70, bmi: 24.2 }],
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
    expect(screen.getByText('Профиль тела')).toBeInTheDocument();
  });

  it('saves with empty optional fields', async () => {
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
    expect(arg.body_fat_percentage).toBe(0);
    expect(arg.muscle_mass_percentage).toBe(0);
    expect(arg.water_percentage).toBe(0);
  });
});
