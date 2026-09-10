import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginForm from './LoginForm';

describe('LoginForm', () => {
  const defaultProps = {
    formData: { email: '', password: '' },
    errors: {},
    generalError: '',
    submitting: false,
    getFieldClass: () => '',
    setField: vi.fn(),
    onSubmit: vi.fn(),
    onSwitchMode: vi.fn(),
  };

  it('renders login form', () => {
    render(<LoginForm {...defaultProps} />);
    expect(screen.getByLabelText('Email')).toBeDefined();
    expect(screen.getByLabelText('Пароль')).toBeDefined();
  });

  it('calls setField on input change', () => {
    const setField = vi.fn();
    render(<LoginForm {...defaultProps} setField={setField} />);
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'test@test.com' } });
    expect(setField).toHaveBeenCalledWith('email', 'test@test.com');
  });

  it('calls onSubmit when form submitted', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn((e) => e.preventDefault());
    render(<LoginForm {...defaultProps} onSubmit={onSubmit} />);
    await user.click(screen.getByText('Войти'));
    expect(onSubmit).toHaveBeenCalled();
  });

  it('calls onSwitchMode when switch button clicked', async () => {
    const user = userEvent.setup();
    const onSwitchMode = vi.fn();
    render(<LoginForm {...defaultProps} onSwitchMode={onSwitchMode} />);
    await user.click(screen.getByText('Создать'));
    expect(onSwitchMode).toHaveBeenCalled();
  });

  it('disables submit when submitting', () => {
    render(<LoginForm {...defaultProps} submitting={true} />);
    expect(screen.getByText('Вход...')).toBeDefined();
  });

  it('shows general error', () => {
    render(<LoginForm {...defaultProps} generalError="Invalid credentials" />);
    expect(screen.getByText('Invalid credentials')).toBeDefined();
  });
});
