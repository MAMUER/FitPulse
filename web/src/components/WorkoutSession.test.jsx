import { act, fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import WorkoutSession from './WorkoutSession';

const t = (ru) => ru;

function renderSession(workout, overrides = {}) {
  const onFinish = overrides.onFinish || vi.fn();
  const onClose = overrides.onClose || vi.fn();
  render(
    <WorkoutSession
      workout={workout}
      t={t}
      onClose={onClose}
      onFinish={onFinish}
    />
  );
  return { onFinish, onClose };
}

describe('WorkoutSession', () => {
  it('renders the title, first exercise and work timer', () => {
    renderSession({ title: 'Силовая', exercises: [['Приседания', '3×12']] });

    expect(screen.getByText('Силовая')).toBeInTheDocument();
    expect(screen.getByText('Приседания')).toBeInTheDocument();
    expect(screen.getByText('0:45')).toBeInTheDocument();
    expect(screen.getByText('работа')).toBeInTheDocument();
  });

  it('toggles the pause state', () => {
    renderSession({ title: 'W', exercises: [['A', 'x']] });

    fireEvent.click(screen.getByText('Пауза'));
    expect(screen.getByText('Продолжить')).toBeInTheDocument();
  });

  it('advances from work to rest via the Next button', () => {
    renderSession({
      title: 'W',
      exercises: [
        ['A', 'x'],
        ['B', 'y'],
      ],
    });

    fireEvent.click(screen.getByText('Далее'));

    expect(screen.getByText('отдых')).toBeInTheDocument();
  });

  it('finishes automatically after the last exercise rest phase', () => {
    const { onFinish, onClose } = renderSession({
      title: 'W',
      exercises: [['A', 'x']],
    });

    fireEvent.click(screen.getByText('Далее')); // work -> rest
    fireEvent.click(screen.getByText('Далее')); // rest -> finish

    expect(onFinish).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it('finishes via the finish button', () => {
    const { onFinish, onClose } = renderSession({
      title: 'W',
      exercises: [['A', 'x']],
    });

    fireEvent.click(screen.getByText('Завершить тренировку'));

    expect(onFinish).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it('renders a workout without exercises without crashing', () => {
    renderSession({ title: 'Empty', exercises: [] });

    expect(screen.getByText('Empty')).toBeInTheDocument();
    expect(screen.getByText('Завершить тренировку')).toBeInTheDocument();
  });

  it('advances from rest to the next exercise via the Next button', () => {
    renderSession({
      title: 'W',
      exercises: [
        ['A', 'x'],
        ['B', 'y'],
      ],
    });

    fireEvent.click(screen.getByText('Далее')); // work -> rest
    expect(screen.getByText('отдых')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Далее')); // rest -> следующее упражнение
    expect(screen.getByText('работа')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
  });

  it('auto-advances work -> rest -> next exercise when the timer expires', () => {
    vi.useFakeTimers();
    try {
      const { onFinish } = renderSession({
        title: 'W',
        exercises: [
          ['A', 'x'],
          ['B', 'y'],
        ],
      });

      // Каждый тик перезапускает таймер от текущего момента, поэтому идём
      // по одной секунде: advanceTimersByTime(45_000) даст только один тик.
      const tick = (seconds) => {
        for (let i = 0; i < seconds; i += 1) {
          act(() => {
            vi.advanceTimersByTime(1000);
          });
        }
      };

      tick(45); // работа истекла — включается отдых
      expect(screen.getByText('отдых')).toBeInTheDocument();

      tick(20); // отдых истёк — следующее упражнение
      expect(screen.getByText('работа')).toBeInTheDocument();
      expect(screen.getByText('B')).toBeInTheDocument();
      expect(onFinish).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it('auto-finishes when the last rest phase expires', () => {
    vi.useFakeTimers();
    try {
      const { onFinish, onClose } = renderSession({
        title: 'W',
        exercises: [['A', 'x']],
      });

      const tick = (seconds) => {
        for (let i = 0; i < seconds; i += 1) {
          act(() => {
            vi.advanceTimersByTime(1000);
          });
        }
      };

      tick(45); // work -> rest
      tick(20); // rest -> finish

      expect(onFinish).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });
});
