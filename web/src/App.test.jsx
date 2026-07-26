import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { AuthProvider, useAuth } from './contexts/AuthContext';

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
    expect(screen.getByText(/fitpulse/i)).toBeInTheDocument();
  });
});

describe('useAuth hook', () => {
  it('returns initial state', () => {
    let result;
    const TestComponent = () => {
      result = useAuth();
      return null;
    };
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );
    expect(result.token).toBeNull();
    expect(result.user).toBeNull();
    expect(result.isAdmin).toBe(false);
    expect(result.loading).toBe(false);
  });
});
