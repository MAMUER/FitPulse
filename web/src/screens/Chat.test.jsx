import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Chat from './Chat';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('Chat', () => {
  it('renders history and sends message', () => {
    const sendChat = vi.fn();
    useApp.mockReturnValue({
      state: {
        chatMessages: [{ id: 1, text: 'hi', sent: false }],
      },
      t: (ru, _en) => ru,
      sendChat,
    });

    render(<Chat />);
    expect(screen.getByText('hi')).toBeInTheDocument();

    const input = screen.getByLabelText('Сообщение');
    fireEvent.change(input, { target: { value: 'hello' } });
    fireEvent.click(screen.getByText('Отправить'));
    expect(sendChat).toHaveBeenCalled();
  });

  it('does not send empty message', () => {
    const sendChat = vi.fn();
    useApp.mockReturnValue({
      state: { chatMessages: [] },
      t: (ru, _en) => ru,
      sendChat,
    });

    render(<Chat />);
    fireEvent.click(screen.getByText('Отправить'));
    expect(sendChat).not.toHaveBeenCalled();
  });
});
