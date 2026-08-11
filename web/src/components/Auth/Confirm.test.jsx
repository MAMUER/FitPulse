import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../../utils/api';
import Confirm from './Confirm';

vi.mock('../../utils/api');

const renderConfirm = (token) => {
  return render(
    <MemoryRouter initialEntries={['/confirm']}>
      <Routes>
        <Route path='/confirm' element={<Confirm token={token} />} />
      </Routes>
    </MemoryRouter>
  );
};

describe('Confirm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows error when no token is provided', () => {
    renderConfirm(null);
    expect(
      screen.getByText(/Токен подтверждения не найден/)
    ).toBeInTheDocument();
  });

  it('shows loading state initially', () => {
    api.confirmEmail.mockImplementation(() => new Promise(() => {}));
    renderConfirm('abc');
    expect(screen.getByText('Пожалуйста, подождите...')).toBeInTheDocument();
  });

  it('shows success message on valid confirmation', async () => {
    api.confirmEmail.mockResolvedValueOnce({});
    renderConfirm('valid-token');

    await waitFor(() => {
      expect(screen.getByText(/Email успешно подтверждён/)).toBeInTheDocument();
    });
  });

  it('shows error message on failed confirmation', async () => {
    api.confirmEmail.mockRejectedValueOnce(new Error('Invalid token'));
    renderConfirm('invalid-token');

    await waitFor(() => {
      expect(screen.getByText('Invalid token')).toBeInTheDocument();
    });
  });

  it('shows back button after confirmation', async () => {
    api.confirmEmail.mockResolvedValueOnce({});
    renderConfirm('valid-token');

    await waitFor(() => {
      expect(screen.getByText('← Вернуться ко входу')).toBeInTheDocument();
    });
  });

  it('navigates back to login when back button is clicked', async () => {
    api.confirmEmail.mockResolvedValueOnce({});
    renderConfirm('valid-token');

    await waitFor(() => {
      expect(screen.getByText('← Вернуться ко входу')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('← Вернуться ко входу'));
  });
});
