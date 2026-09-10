import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import VerifyEmail from './VerifyEmail';

describe('VerifyEmail', () => {
  it('renders verify email form', () => {
    render(<VerifyEmail email="test@test.com" successMessage="" generalError="" onBack={vi.fn()} />);
    expect(screen.getByText('Проверьте почту')).toBeDefined();
    expect(screen.getByText('test@test.com')).toBeDefined();
  });

  it('shows fallback email when none provided', () => {
    render(<VerifyEmail email="" successMessage="" generalError="" onBack={vi.fn()} />);
    expect(screen.getByText('ваш email')).toBeDefined();
  });

  it('shows success message', () => {
    render(<VerifyEmail email="test@test.com" successMessage="Check your email" generalError="" onBack={vi.fn()} />);
    expect(screen.getByText('Check your email')).toBeDefined();
  });

  it('shows general error', () => {
    render(<VerifyEmail email="test@test.com" successMessage="" generalError="Something went wrong" onBack={vi.fn()} />);
    expect(screen.getByText('Something went wrong')).toBeDefined();
  });

  it('calls onBack when back button clicked', async () => {
    const user = userEvent.setup();
    const onBack = vi.fn();
    render(<VerifyEmail email="test@test.com" successMessage="" generalError="" onBack={onBack} />);
    await user.click(screen.getByText('← Вернуться ко входу'));
    expect(onBack).toHaveBeenCalled();
  });
});
