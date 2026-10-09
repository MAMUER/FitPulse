import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Reset from './Reset';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'reset',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    resetStep: 0,
    resetEmail: '',
  },
  t: (ru, _en) => ru,
  submitResetEmail: vi.fn(),
  submitResetCode: vi.fn(),
  submitNewPassword: vi.fn(),
  forgotPassword: vi.fn(),
  go: vi.fn(),
};

function renderReset() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Reset />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Reset', () => {
  it('renders reset form step 1', () => {
    renderReset();
    expect(screen.getByText('Восстановление')).toBeInTheDocument();
  });
});
