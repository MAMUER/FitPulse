import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import Terms from './Terms';

describe('Terms', () => {
  it('renders terms heading', () => {
    render(<Terms />);
    expect(
      screen.getByRole('heading', {
        name: 'Пользовательское соглашение — FitPulse',
      })
    ).toBeInTheDocument();
  });

  it('sets document title', async () => {
    render(<Terms />);
    await waitFor(() => {
      expect(document.title).toBe('Пользовательское соглашение — FitPulse');
    });
  });

  it('renders update date', () => {
    render(<Terms />);
    expect(screen.getByText(/2026-07-29/)).toBeInTheDocument();
  });

  it('renders contact email', () => {
    render(<Terms />);
    expect(screen.getByText('mihnikolaenko12@yandex.ru')).toBeInTheDocument();
  });
});
