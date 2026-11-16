import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Videos from './Videos';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

const trainingVideos = [
  {
    id: 'v1',
    title: 'Утренняя зарядка',
    description: 'Лёгкий комплекс для старта дня.',
    image: '/video1.jpg',
    duration: '12 мин',
    level: 'Лёгкий',
    equipment: ['Коврик'],
  },
  {
    id: 'v2',
    title: 'Силовая тренировка',
    description: 'Базовые упражнения с гантелями.',
    image: '/video2.jpg',
    duration: '25 мин',
    level: 'Средний',
    equipment: ['Гантели'],
  },
];

function makeContext(overrides = {}) {
  const { state: stateOverrides, ...rest } = overrides;
  return {
    state: {
      trainingVideos,
      videos: [],
      ...stateOverrides,
    },
    t: (ru, _en) => ru,
    loadVideos: vi.fn(),
    notify: vi.fn(),
    ...rest,
  };
}

describe('Videos', () => {
  it('falls back to the index when a video has no id', () => {
    const context = makeContext({
      state: { trainingVideos: [{ title: 'Без id', image: '/x.jpg' }] },
    });
    useApp.mockReturnValue(context);
    const { container } = render(<Videos />);

    expect(container.querySelectorAll('.video-card')).toHaveLength(1);
    expect(screen.getByText('Без id')).toBeInTheDocument();
  });

  it('renders library rows without id and falls back on duration', () => {
    const context = makeContext({
      state: {
        videos: [
          { title: 'Только минуты', minutes: 10 },
          { title: 'Без длительности' },
          { id: 'lib1', title: 'С длительностью', duration: '20' },
        ],
      },
    });
    useApp.mockReturnValue(context);
    const { container } = render(<Videos />);

    expect(container.querySelectorAll('.body-row')).toHaveLength(3);
    expect(container.querySelectorAll('.body-row')[0].textContent).toContain(
      'Только минуты'
    );
    expect(container.querySelectorAll('.body-row')[1].textContent).toContain(
      'Без длительности'
    );
    expect(container.textContent).toContain('10');
    expect(container.textContent).toContain('20');
  });

  it('loads and renders video lessons', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Videos />);

    expect(context.loadVideos).toHaveBeenCalled();
    expect(screen.getByText('Утренняя зарядка')).toBeInTheDocument();
    expect(screen.getByText(/12 мин/)).toBeInTheDocument();
    expect(screen.getByText('Силовая тренировка')).toBeInTheDocument();
  });

  it('renders equipment cards', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Videos />);

    expect(screen.getByText('Коврик')).toBeInTheDocument();
    expect(screen.getByText('Гантели')).toBeInTheDocument();
    expect(screen.getByText('Резинка')).toBeInTheDocument();
    expect(screen.getByText('Таймер')).toBeInTheDocument();
  });

  it('opens a video modal and closes it', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Videos />);
    fireEvent.click(screen.getByText('Утренняя зарядка'));

    const dialog = screen.getByRole('dialog');
    expect(
      within(dialog).getByRole('heading', { name: 'Утренняя зарядка' })
    ).toBeInTheDocument();
    fireEvent.click(within(dialog).getByLabelText('Закрыть'));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('switches between videos in the modal', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Videos />);
    fireEvent.click(screen.getByText('Утренняя зарядка'));
    fireEvent.click(screen.getByLabelText('Следующее видео'));

    expect(
      within(screen.getByRole('dialog')).getByRole('heading', {
        name: 'Силовая тренировка',
      })
    ).toBeInTheDocument();
  });

  it('renders the FitPulse library when available', () => {
    const context = makeContext({
      state: {
        videos: [
          { id: 1, title: 'Растяжка', duration: 15 },
          { id: 2, title: 'Кардио', minutes: 20 },
        ],
      },
    });
    useApp.mockReturnValue(context);

    render(<Videos />);

    expect(screen.getByText('Библиотека FitPulse')).toBeInTheDocument();
    expect(screen.getByText('Растяжка')).toBeInTheDocument();
    expect(screen.getByText('Кардио')).toBeInTheDocument();
  });

  it('does not render the library section without videos', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Videos />);

    expect(screen.queryByText('Библиотека FitPulse')).not.toBeInTheDocument();
  });

  it('handles missing video collections', () => {
    const context = makeContext({
      state: { trainingVideos: null, videos: null },
    });
    useApp.mockReturnValue(context);

    render(<Videos />);

    expect(screen.getByText('Оборудование для дома')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
