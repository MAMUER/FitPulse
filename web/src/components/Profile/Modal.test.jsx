import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import Modal from './Modal';

describe('Modal', () => {
  it('renders modal content', () => {
    render(
      <Modal onClose={vi.fn()} ariaLabel='Test Modal'>
        <div>Modal Content</div>
      </Modal>
    );
    expect(screen.getByText('Modal Content')).toBeDefined();
  });

  it('calls onClose when escape pressed', () => {
    const onClose = vi.fn();
    render(
      <Modal onClose={onClose} ariaLabel='Test Modal'>
        <div>Modal Content</div>
      </Modal>
    );
    const event = new KeyboardEvent('keydown', { key: 'Escape' });
    document.dispatchEvent(event);
    expect(onClose).toHaveBeenCalled();
  });

  it('renders with aria-label', () => {
    render(
      <Modal onClose={vi.fn()} ariaLabel='Test Modal'>
        <div>Modal Content</div>
      </Modal>
    );
    const dialog = document.querySelector('dialog');
    expect(dialog).toHaveAttribute('aria-label', 'Test Modal');
  });
});
