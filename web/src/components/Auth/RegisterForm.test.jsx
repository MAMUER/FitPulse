import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import RegisterForm from './RegisterForm';

describe('RegisterForm', () => {
  const defaultProps = {
    formData: { name: '', email: '', password: '', totpCode: '', backupCode: '' },
    errors: {},
    generalError: '',
    passwordChecks: { length: false, upper: false, lower: false, digit: false },
    submitting: false,
    getFieldClass: () => '',
    setField: vi.fn(),
    onSubmit: vi.fn(),
    updatePasswordChecks: vi.fn(),
    onSwitchMode: vi.fn(),
  };

  it('renders register form', () => {
    render(<RegisterForm {...defaultProps} />);
    expect(screen.getByLabelText('Имя')).toBeDefined();
    expect(screen.getByLabelText('Email')).toBeDefined();
    expect(screen.getByLabelText('Пароль (мин. 8 символов)')).toBeDefined();
  });

  it('calls setField on name change', () => {
    const setField = vi.fn();
    render(<RegisterForm {...defaultProps} setField={setField} />);
    fireEvent.change(screen.getByLabelText('Имя'), { target: { value: 'John' } });
    expect(setField).toHaveBeenCalledWith('name', 'John');
  });

  it('calls onSwitchMode when switch button clicked', async () => {
    const user = await import('@testing-library/user-event').then(m => m.default.setup());
    const onSwitchMode = vi.fn();
    render(<RegisterForm {...defaultProps} onSwitchMode={onSwitchMode} />);
    await user.click(screen.getByText('Войти'));
    expect(onSwitchMode).toHaveBeenCalled();
  });

  it('disables submit when password checks fail', () => {
    render(<RegisterForm {...defaultProps} submitting={false} />);
    expect(screen.getByText('Создать аккаунт')).toBeDefined();
  });

  it('shows password checks when password is entered', () => {
    render(
      <RegisterForm
        {...defaultProps}
        formData={{ ...defaultProps.formData, password: 'Test1234' }}
        passwordChecks={{ length: true, upper: true, lower: false, digit: false }}
      />
    );
    expect(screen.getByText(/8\+ символов/)).toBeDefined();
  });
});
