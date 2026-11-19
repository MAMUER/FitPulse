import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import VideoModal from './VideoModal';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

const VIDEOS = [
  {
    id: 1,
    title: 'Приседания',
    level: 'Средний',
    duration: '12 мин',
    image: '/ squat.jpg',
    description: 'Техника приседаний',
    equipment: ['Коврик'],
  },
  {
    id: 2,
    title: 'Планка',
    level: 'Лёгкий',
    duration: '5 мин',
    image: '/plank.jpg',
  },
];

function setup(overrides = {}) {
  const notify = vi.fn();
  const onClose = vi.fn();
  const onChange = vi.fn();
  useApp.mockReturnValue({
    state: {},
    t: (ru, _en) => ru,
    notify,
    ...overrides,
  });
  return { notify, onClose, onChange };
}

describe('VideoModal', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('renders nothing when the index is out of range', () => {
    setup();
    const { container } = render(
      <VideoModal
        videos={VIDEOS}
        index={5}
        onChange={vi.fn()}
        onClose={vi.fn()}
      />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('renders the selected video', () => {
    setup();
    render(
      <VideoModal
        videos={VIDEOS}
        index={0}
        onChange={vi.fn()}
        onClose={vi.fn()}
      />
    );
    expect(screen.getByText('Приседания')).toBeInTheDocument();
    expect(screen.getByText('Техника приседаний')).toBeInTheDocument();
    expect(screen.getByText('Коврик')).toBeInTheDocument();
  });

  it('renders a video without optional fields', () => {
    setup();
    render(
      <VideoModal
        videos={VIDEOS}
        index={1}
        onChange={vi.fn()}
        onClose={vi.fn()}
      />
    );
    expect(screen.getByText('Планка')).toBeInTheDocument();
  });

  it('notifies instead of playing, because the player is external', () => {
    const { notify } = setup();
    render(
      <VideoModal
        videos={VIDEOS}
        index={0}
        onChange={vi.fn()}
        onClose={vi.fn()}
      />
    );
    fireEvent.click(screen.getByText('Открыть видео'));
    expect(notify).toHaveBeenCalledWith(
      expect.stringContaining('Видео пока недоступно')
    );
  });

  it('closes via the close button', () => {
    const { onClose } = setup();
    render(
      <VideoModal
        videos={VIDEOS}
        index={0}
        onChange={vi.fn()}
        onClose={onClose}
      />
    );
    fireEvent.click(screen.getByLabelText('Закрыть'));
    expect(onClose).toHaveBeenCalled();
  });

  it('switches to the previous and next video', () => {
    const { onChange } = setup();
    render(
      <VideoModal
        videos={VIDEOS}
        index={1}
        onChange={onChange}
        onClose={vi.fn()}
      />
    );
    fireEvent.click(screen.getByLabelText('Предыдущее видео'));
    expect(onChange).toHaveBeenLastCalledWith(-1);
    fireEvent.click(screen.getByLabelText('Следующее видео'));
    expect(onChange).toHaveBeenLastCalledWith(1);
  });
});
