import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Toast from './Toast';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('Toast', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('renders nothing when there is no toast message', () => {
    useApp.mockReturnValue({ state: { __toast: null } });
    const { container } = render(<Toast />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders the toast message when set', () => {
    useApp.mockReturnValue({ state: { __toast: 'Фото удалено' } });
    render(<Toast />);
    expect(screen.getByText('Фото удалено')).toBeInTheDocument();
    expect(screen.getByRole('status')).toBeInTheDocument();
  });
});
