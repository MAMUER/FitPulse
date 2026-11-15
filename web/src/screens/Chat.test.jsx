import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Chat from './Chat';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

const makeContext = (overrides = {}) => ({
  state: {
    guest: true,
    chatSettings: {
      onlineStatus: false,
      readReceipts: true,
      notifications: true,
    },
    chatContacts: [],
    chatMessages: [],
    ...overrides.state,
  },
  t: (ru) => ru,
  go: vi.fn(),
  openLegal: vi.fn(),
  openChat: vi.fn(),
  openFriendProfile: vi.fn(),
  saveChatSettings: vi.fn(),
  sendChat: vi.fn(),
  ...overrides,
});

describe('Chat', () => {
  it('shows the reference guest restriction and opens local settings', async () => {
    const user = userEvent.setup();
    useApp.mockReturnValue(makeContext());
    render(<Chat />);

    expect(
      screen.getByText('Личные сообщения недоступны гостю')
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Настройки чата' }));

    expect(screen.getByRole('dialog', { name: 'Настройки чата' })).toBeVisible();
    expect(screen.getByLabelText('Показывать онлайн-статус')).toBeInTheDocument();
  });

  it('persists settings and closes the modal', async () => {
    const user = userEvent.setup();
    const context = makeContext();
    useApp.mockReturnValue(context);
    render(<Chat />);

    await user.click(screen.getByRole('button', { name: 'Настройки чата' }));
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));

    expect(context.saveChatSettings).toHaveBeenCalledTimes(1);
    expect(
      screen.queryByRole('dialog', { name: 'Настройки чата' })
    ).not.toBeInTheDocument();
  });

  it('shows online contacts and opens their conversation for registered users', async () => {
    const user = userEvent.setup();
    const openChat = vi.fn();
    useApp.mockReturnValue(
      makeContext({
        state: {
          guest: false,
          onlineUsers: 1,
          chatContacts: [
            {
              name: 'Анна',
              last: 'До встречи',
              time: '12:30',
              unread: 2,
              avatar: '/anna.jpg',
              online: true,
              status: 'gym',
            },
          ],
        },
        openChat,
      })
    );
    render(<Chat />);

    expect(screen.getByText('До встречи')).toBeInTheDocument();
    await user.click(screen.getAllByRole('button', { name: /Анна/ })[0]);

    expect(openChat).toHaveBeenCalledWith('Анна');
  });

  it('sends a non-empty message from a selected conversation', () => {
    const sendChat = vi.fn();
    useApp.mockReturnValue(
      makeContext({
        state: {
          guest: false,
          chatConsent: true,
          selectedChat: 'Анна',
          chatContacts: [{ name: 'Анна', avatar: '/anna.jpg' }],
          chatMessages: [{ text: 'Привет', time: '12:20', sent: false }],
        },
        sendChat,
      })
    );
    render(<Chat />);

    const input = screen.getByLabelText('Сообщение');
    fireEvent.change(input, { target: { value: 'Как дела?' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }));

    expect(screen.getByText('Привет')).toBeInTheDocument();
    expect(sendChat).toHaveBeenCalledTimes(1);
  });

  it('does not send an empty message', () => {
    const sendChat = vi.fn();
    useApp.mockReturnValue(
      makeContext({
        state: {
          guest: false,
          chatConsent: true,
          selectedChat: 'Анна',
          chatContacts: [],
        },
        sendChat,
      })
    );
    render(<Chat />);

    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }));

    expect(sendChat).not.toHaveBeenCalled();
  });
});
