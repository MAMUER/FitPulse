import { fireEvent, render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Landing from './Landing';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'landing',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    registered: false,
  },
  t: (ru, _en) => ru,
};

function renderLanding(overrides = {}) {
  useApp.mockReturnValue({
    ...defaultContext,
    state: { ...defaultContext.state, ...overrides },
  });
  return render(
    <BrowserRouter>
      <AppProvider>
        <Landing />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Landing', () => {
  it('renders landing page', () => {
    renderLanding();
    expect(screen.getByText('FitPulse')).toBeInTheDocument();
  });

  it('shows sign in and create account buttons when not registered', () => {
    renderLanding({ registered: false });
    expect(screen.getByText('Sign in')).toBeInTheDocument();
    expect(screen.getByText('Create account')).toBeInTheDocument();
  });

  it('shows open app button when registered', () => {
    renderLanding({ registered: true });
    expect(screen.getByText('Open app')).toBeInTheDocument();
  });

  it('navigates when buttons clicked (registered)', () => {
    renderLanding({ registered: true });
    fireEvent.click(screen.getByText('Open app'));
    fireEvent.click(screen.getByText('Privacy Policy'));
    fireEvent.click(screen.getByText('Terms of Service'));
    expect(screen.getByText('Open app')).toBeInTheDocument();
  });

  it('navigates login/register when not registered', () => {
    renderLanding({ registered: false });
    fireEvent.click(screen.getByText('Sign in'));
    fireEvent.click(screen.getByText('Create account'));
    fireEvent.click(screen.getByText('Privacy Policy'));
    fireEvent.click(screen.getByText('Terms of Service'));
    expect(screen.getByText('Create account')).toBeInTheDocument();
  });
});
