import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import TwoFAForm from './TwoFAForm';

describe('TwoFAForm', () => {
  const defaultProps = {
    formData: { totpCode: '', backupCode: '' },
    generalError: '',
    submitting: false,
    setField: vi.fn(),
    onSubmit: vi.fn(),
    onBack: vi.fn(),
  };

  it('renders 2FA form', () => {
    render(<TwoFAForm {...defaultProps} />);
    expect(screen.getByLabelText('6-значный код')).toBeDefined();
    expect(screen.getByLabelText('Резервный код xxxx-xxxx')).toBeDefined();
  });

  it('calls setField on TOTP code change', async () => {
    const setField = vi.fn();
    render(<TwoFAForm {...defaultProps} setField={setField} />);
    fireEvent.change(screen.getByLabelText('6-значный код'), {
      target: { value: '123456' },
    });
    expect(setField).toHaveBeenCalledWith('totpCode', '123456');
  });

  it('calls onBack when back button clicked', async () => {
    const user = userEvent.setup();
    const onBack = vi.fn();
    render(<TwoFAForm {...defaultProps} onBack={onBack} />);
    await user.click(screen.getByText('← Вернуться ко входу'));
    expect(onBack).toHaveBeenCalled();
  });

  it('uses backup code when button clicked', async () => {
    const user = userEvent.setup();
    const setField = vi.fn();
    render(
      <TwoFAForm
        {...defaultProps}
        setField={setField}
        formData={{ totpCode: '123456', backupCode: '' }}
      />
    );
    await user.click(screen.getByText('Использовать резервный код'));
    expect(setField).toHaveBeenCalledWith('backupCode', '123456');
  });

  it('shows general error', () => {
    render(<TwoFAForm {...defaultProps} generalError='Invalid code' />);
    const errors = screen.getAllByText('Invalid code');
    expect(errors.length).toBeGreaterThanOrEqual(1);
  });
});
