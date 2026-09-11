import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import Privacy from './Privacy';

describe('Privacy', () => {
  it('renders privacy page', () => {
    render(<Privacy />);
    expect(
      screen.getByText('Политика конфиденциальности — FitPulse')
    ).toBeDefined();
  });

  it('renders policy sections', () => {
    render(<Privacy />);
    expect(screen.getByText('1. Общие положения')).toBeDefined();
    expect(screen.getByText('2. Какие данные мы собираем')).toBeDefined();
  });
});
