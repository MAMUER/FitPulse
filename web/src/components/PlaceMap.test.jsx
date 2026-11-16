import { fireEvent, render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import PlaceMap from './PlaceMap';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

const PINNED = [
  { name: 'Зал', x: 40, y: 30, color: '#6fae20', icon: 'fa-dumbbell' },
  {
    name: 'Бассейн',
    x: 60,
    y: 50,
    color: '#19d8da',
    icon: 'fa-person-swimming',
  },
];

function setup(state = {}) {
  const notify = vi.fn();
  const selectPlace = vi.fn();
  const addPlace = vi.fn();
  const editPlace = vi.fn();
  useApp.mockReturnValue({
    state: { pinnedPlaces: PINNED, selectedPlace: 'Дом', ...state },
    t: (ru, _en) => ru,
    notify,
    selectPlace,
    addPlace,
    editPlace,
  });
  return { notify, selectPlace, addPlace, editPlace };
}

describe('PlaceMap', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('renders the map, the place buttons and the pinned list', () => {
    setup();
    render(<PlaceMap />);
    expect(screen.getByTitle('Карта тренировок')).toBeInTheDocument();
    expect(screen.getByText('Закреплённые точки на карте')).toBeInTheDocument();
    expect(screen.getByText('Добавить точку')).toBeInTheDocument();
    // The pin carries the title attribute; the place button does not.
    expect(screen.getByTitle('Зал')).toBeInTheDocument();
  });

  it('falls back to the default place and an empty pin list', () => {
    setup({ pinnedPlaces: undefined, selectedPlace: undefined });
    render(<PlaceMap />);
    const placeButtons = document.querySelector('.place-buttons');
    // "Дом" also appears in the "Выбрано" summary, so scope to the buttons.
    expect(
      within(placeButtons).getByRole('button', { name: /Дом/ })
    ).toBeInTheDocument();
    expect(document.querySelector('.fp-map-pin')).toBeNull();
  });

  it('selects a place and adds a new pin', () => {
    const { selectPlace, addPlace } = setup();
    render(<PlaceMap />);

    const placeButtons = document.querySelector('.place-buttons');
    fireEvent.click(
      within(placeButtons).getByRole('button', { name: /Бассейн/ })
    );
    expect(selectPlace).toHaveBeenCalledWith('Бассейн');

    fireEvent.click(screen.getByRole('button', { name: /Добавить точку/ }));
    expect(addPlace).toHaveBeenCalled();
  });

  it('edits the position of a pinned place', () => {
    const { editPlace } = setup();
    render(<PlaceMap />);
    fireEvent.click(screen.getByTitle('Зал'));
    expect(editPlace).toHaveBeenCalledWith(0);
  });

  it('opens the external map and explains why', () => {
    const { notify } = setup();
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    render(<PlaceMap />);

    fireEvent.click(screen.getByText('Открыть карту'));
    expect(notify).toHaveBeenCalledWith(
      'Интерактивная карта и геокодирование требуют картографического API. Открыт внешний просмотр.'
    );
    expect(openSpy).toHaveBeenCalledWith(
      'https://yandex.ru/maps',
      '_blank',
      'noopener'
    );
  });
});
