import { fireEvent, render, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import PlaceMap from './PlaceMap';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

function setup(state = {}) {
  const selectPlace = vi.fn();
  useApp.mockReturnValue({
    state: { selectedPlace: 'Дом', ...state },
    t: (ru, _en) => ru,
    selectPlace,
  });
  return { selectPlace };
}

describe('PlaceMap', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('renders all place buttons with the default active', () => {
    setup();
    render(<PlaceMap />);
    const container = document.querySelector('.place-buttons');
    expect(container).not.toBeNull();
    const scope = within(container);
    for (const name of ['Дом', 'Работа', 'Бассейн', 'Зал', 'Учеба']) {
      expect(
        scope.getByRole('button', { name: new RegExp(name) })
      ).toBeDefined();
    }
    // По умолчанию активна «Дом».
    expect(scope.getByRole('button', { name: /Дом/ }).className).toContain(
      'active'
    );
  });

  it('selects a place', () => {
    const { selectPlace } = setup();
    render(<PlaceMap />);
    const scope = within(document.querySelector('.place-buttons'));

    fireEvent.click(scope.getByRole('button', { name: /Зал/ }));
    expect(selectPlace).toHaveBeenCalledWith('Зал');
  });

  it('marks the selected place as active', () => {
    setup({ selectedPlace: 'Бассейн' });
    render(<PlaceMap />);
    const scope = within(document.querySelector('.place-buttons'));
    expect(scope.getByRole('button', { name: /Бассейн/ }).className).toContain(
      'active'
    );
  });
});
