import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DeleteProfileModal from './DeleteProfileModal';

const mockUseAuth = vi.fn();
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}));

const mockDeleteProfile = vi.fn();
vi.mock('../../utils/api', () => ({
  deleteProfile: () => mockDeleteProfile(),
}));

describe('DeleteProfileModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({ logout: vi.fn() });
  });

  it('renders delete profile form', () => {
    render(<DeleteProfileModal onClose={vi.fn()} />);
    expect(
      screen.getByLabelText('Введите пароль для подтверждения')
    ).toBeDefined();
  });

  it('shows error when password is empty', async () => {
    render(<DeleteProfileModal onClose={vi.fn()} />);
    const form = document.querySelector('form');
    fireEvent.submit(form);
    expect(screen.getByText(/Введите пароль/i)).toBeDefined();
  });

  it('calls logout and redirects on successful delete', async () => {
    const user = userEvent.setup();
    const logout = vi.fn();
    mockUseAuth.mockReturnValue({ logout });
    mockDeleteProfile.mockResolvedValue({});
    render(<DeleteProfileModal onClose={vi.fn()} />);
    await user.type(
      screen.getByLabelText('Введите пароль для подтверждения'),
      'password'
    );
    const form = document.querySelector('form');
    fireEvent.submit(form);
    await waitFor(() => expect(logout).toHaveBeenCalled());
  });
});
