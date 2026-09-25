import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Integrations from './Integrations';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'integrations',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
  },
  t: (ru, _en) => ru,
  notify: vi.fn(),
  loadIntegrationProviders: vi.fn().mockResolvedValue([]),
  disconnectIntegration: vi.fn(),
};

function renderIntegrations(overrides = {}) {
  useApp.mockReturnValue({
    ...defaultContext,
    state: { ...defaultContext.state, ...overrides },
  });
  return render(
    <BrowserRouter>
      <AppProvider>
        <Integrations />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Integrations', () => {
  it('renders integrations screen', () => {
    renderIntegrations();
    expect(screen.getByText('Интеграции')).toBeInTheDocument();
  });

  it('shows loading state initially', () => {
    renderIntegrations();
    expect(screen.getByText('Загрузка...')).toBeInTheDocument();
  });
});
