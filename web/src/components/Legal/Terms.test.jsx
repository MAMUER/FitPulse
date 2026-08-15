import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import Terms from './Terms';

describe('Terms', () => {
  it('renders terms heading and content', () => {
    render(<Terms />);
    expect(
      screen.getByRole('heading', {
        name: 'Пользовательское соглашение — FitPulse',
      })
    ).toBeInTheDocument();
    expect(screen.getByText(/2026-07-29/)).toBeInTheDocument();
    expect(screen.getByText(/mihnikolaenko12@yandex.ru/)).toBeInTheDocument();
  });

  it('sets document title on mount', async () => {
    render(<Terms />);
    await waitFor(() => {
      expect(document.title).toBe('Пользовательское соглашение — FitPulse');
    });
  });
});
