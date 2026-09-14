import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Confirm from './Confirm';

const mockApiConfirmEmail = vi.fn();
vi.mock('../../utils/api', () => ({
  confirmEmail: () => mockApiConfirmEmail(),
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  };
});

describe('Confirm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows error when no token', () => {
    render(<Confirm token={null} />);
    expect(screen.getByText(/Токен подтверждения не найден/i)).toBeDefined();
  });

  it('confirms email successfully', async () => {
    mockApiConfirmEmail.mockResolvedValue({});
    render(<Confirm token='valid-token' />);
    expect(await screen.findByText(/Email успешно подтверждён/i)).toBeDefined();
  });

  it('shows error on failed confirmation', async () => {
    mockApiConfirmEmail.mockRejectedValue(new Error('Invalid token'));
    render(<Confirm token='invalid-token' />);
    expect(await screen.findByText('Invalid token')).toBeDefined();
  });

  it('renders back button after confirmation', async () => {
    mockApiConfirmEmail.mockResolvedValue({});
    render(<Confirm token='valid-token' />);
    expect(await screen.findByText('← Вернуться ко входу')).toBeDefined();
  });
});
