import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import Privacy from './Privacy';

describe('Privacy', () => {
  it('renders privacy heading', () => {
    render(<Privacy />);
    expect(
      screen.getByRole('heading', {
        name: 'Политика конфиденциальности — FitPulse',
      })
    ).toBeInTheDocument();
  });

  it('sets document title', async () => {
    render(<Privacy />);
    await waitFor(() => {
      expect(document.title).toBe('Политика конфиденциальности — FitPulse');
    });
  });

  it('renders update date', () => {
    render(<Privacy />);
    expect(screen.getByText(/2026-07-29/)).toBeInTheDocument();
  });

  it('renders contact email', () => {
    render(<Privacy />);
    expect(
      screen.getAllByText('mihnikolaenko12@yandex.ru').length
    ).toBeGreaterThanOrEqual(1);
  });
});
