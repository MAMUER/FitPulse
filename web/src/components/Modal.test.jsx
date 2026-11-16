import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import Modal from './Modal';

function setup(props = {}) {
  const onClose = vi.fn();
  const utils = render(
    <Modal label='Тестовая модалка' onClose={onClose} {...props}>
      <p>Содержимое</p>
    </Modal>
  );
  return { onClose, ...utils };
}

describe('Modal', () => {
  it('renders the dialog with an accessible name', () => {
    setup();
    expect(
      screen.getByRole('dialog', { name: 'Тестовая модалка' })
    ).toBeVisible();
    expect(screen.getByText('Содержимое')).toBeInTheDocument();
  });

  it('closes on a backdrop click', () => {
    const { onClose } = setup();
    fireEvent.click(document.querySelector('.fp-modal-backdrop'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('closes on Escape', () => {
    const { onClose } = setup();
    const dialog = screen.getByRole('dialog');
    fireEvent.keyDown(dialog, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('ignores other keys', () => {
    const { onClose } = setup();
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Enter' });
    expect(onClose).not.toHaveBeenCalled();
  });

  it('detaches the Escape listener on unmount', () => {
    const onClose = vi.fn();
    const { unmount } = render(
      <Modal label='Модалка' onClose={onClose}>
        <p>Содержимое</p>
      </Modal>
    );
    unmount();
    expect(onClose).not.toHaveBeenCalled();
  });
});
