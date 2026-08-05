import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { AuthProvider } from './contexts/AuthContext';

vi.mock('./utils/api', () => ({
  fetchUserProfile: vi.fn(),
  fetchTodayWorkout: vi.fn(),
  fetchAIRecommendations: vi.fn(),
  fetchStats: vi.fn(),
}));

const renderApp = () => {
  return render(
    <BrowserRouter>
      <AuthProvider>
        <App />
      </AuthProvider>
    </BrowserRouter>
  );
};

describe('App Router', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders without crashing', () => {
    renderApp();
    expect(screen.getAllByText(/fitpulse/i).length).toBeGreaterThan(0);
  });

  it('shows auth screen when not authenticated', async () => {
    renderApp();
    const loginText = await screen.findByText(/Войти/i, {}, { timeout: 3000 });
    expect(loginText).toBeInTheDocument();
  });
});
