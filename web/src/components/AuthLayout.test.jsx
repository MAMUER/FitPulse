import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import AuthLayout from './AuthLayout';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('AuthLayout', () => {
  it('renders children and toggles', () => {
    const toggleHighContrast = vi.fn();
    const toggleLanguage = vi.fn();
    useApp.mockReturnValue({
      state: { language: 'ru' },
      t: (ru, en) => (ru ? ru : en),
      toggleHighContrast,
      toggleLanguage,
    });

    render(
      <AuthLayout>
        <div>test-child</div>
      </AuthLayout>
    );

    expect(screen.getByText('FitPulse')).toBeInTheDocument();
    expect(screen.getByText('test-child')).toBeInTheDocument();
    fireEvent.click(
      screen.getByText(/High contrast mode|Режим высокой контрастности/i)
    );
    expect(toggleHighContrast).toHaveBeenCalled();
    fireEvent.click(screen.getByText('EN'));
    expect(toggleLanguage).toHaveBeenCalled();
  });

  it('shows RU when language is en', () => {
    useApp.mockReturnValue({
      state: { language: 'en' },
      t: (_ru, en) => en,
      toggleHighContrast: vi.fn(),
      toggleLanguage: vi.fn(),
    });

    render(<AuthLayout>child</AuthLayout>);
    expect(screen.getByText('RU')).toBeInTheDocument();
  });
});
