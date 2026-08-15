import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import Privacy from './Privacy';

describe('Privacy', () => {
  it('renders privacy heading and content', () => {
    render(<Privacy />);
    expect(
      screen.getByRole('heading', {
        name: 'Политика конфиденциальности — FitPulse',
      })
    ).toBeInTheDocument();
    expect(screen.getByText(/2026-07-29/)).toBeInTheDocument();
    expect(
      screen.getAllByText(/mihnikolaenko12@yandex.ru/).length
    ).toBeGreaterThanOrEqual(1);
  });

  it('sets document title on mount', async () => {
    render(<Privacy />);
    await waitFor(() => {
      expect(document.title).toBe('Политика конфиденциальности — FitPulse');
    });
  });
});
