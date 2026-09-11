import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import Terms from './Terms';

describe('Terms', () => {
  it('renders terms page', () => {
    render(<Terms />);
    expect(
      screen.getByText('Пользовательское соглашение — FitPulse')
    ).toBeDefined();
  });

  it('renders terms sections', () => {
    render(<Terms />);
    expect(screen.getByText('1. Принятие условий')).toBeDefined();
    expect(screen.getByText('2. Описание сервиса')).toBeDefined();
  });
});
