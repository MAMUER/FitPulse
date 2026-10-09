import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Confirm from './Confirm';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'confirm',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
  },
  t: (ru, _en) => ru,
  notify: vi.fn(),
};

function renderConfirm() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Confirm />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Confirm', () => {
  it('renders confirmation screen', () => {
    renderConfirm();
    expect(screen.getByText('FitPulse')).toBeInTheDocument();
  });
});
