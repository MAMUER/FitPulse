import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ChangeEmailModal from './ChangeEmailModal';

const mockChangeEmail = vi.fn();
vi.mock('../../utils/api', () => ({
  changeEmail: () => mockChangeEmail(),
}));

describe('ChangeEmailModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders change email form', () => {
    render(<ChangeEmailModal onClose={vi.fn()} />);
    expect(screen.getByLabelText('Новый email')).toBeDefined();
    expect(screen.getByLabelText('Текущий пароль')).toBeDefined();
  });

  it('shows error when fields are empty', async () => {
    render(<ChangeEmailModal onClose={vi.fn()} />);
    const form = document.querySelector('form');
    fireEvent.submit(form);
    expect(screen.getByText(/Заполните все поля/i)).toBeDefined();
  });

  it('calls onClose on successful change', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    mockChangeEmail.mockResolvedValue({});
    render(<ChangeEmailModal onClose={onClose} />);
    await user.type(screen.getByLabelText('Новый email'), 'new@test.com');
    await user.type(screen.getByLabelText('Текущий пароль'), 'password');
    const form = document.querySelector('form');
    fireEvent.submit(form);
    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });
});
