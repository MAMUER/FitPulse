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

    expect(
      screen.getByRole('dialog', { name: 'Настройки чата' })
    ).toBeVisible();
    expect(
      screen.getByLabelText('Показывать онлайн-статус')
    ).toBeInTheDocument();
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

  it('guest view links to sign in, register and the legal pages', () => {
    const context = makeContext({ state: { guest: true } });
    useApp.mockReturnValue(context);
    render(<Chat />);

    fireEvent.click(screen.getByText('Войти'));
    expect(context.go).toHaveBeenCalledWith('login');

    fireEvent.click(screen.getByText('Зарегистрироваться'));
    expect(context.go).toHaveBeenCalledWith('register');

    for (const [label, page] of [
      ['Политика конфиденциальности', 'privacy'],
      ['Соглашение', 'terms'],
      ['Персональные данные', 'consent'],
    ]) {
      fireEvent.click(screen.getByText(label));
      expect(context.openLegal).toHaveBeenLastCalledWith(page);
    }
  });

  it('contacts view opens a chat and the settings dialog', () => {
    const context = makeContext({
      state: {
        guest: false,
        chatContacts: [
          {
            name: 'Анна',
            avatar: '/anna.jpg',
            online: true,
            status: 'gym',
            last: 'Привет',
            time: '12:00',
            unread: 0,
          },
          {
            name: 'Иван',
            avatar: '/ivan.jpg',
            online: true,
            status: 'weekend',
            last: 'Ок',
            time: '13:00',
          },
        ],
      },
    });
    useApp.mockReturnValue(context);
    render(<Chat />);

    // "weekend" is not a known status, so getStatusClass returns ''.
    fireEvent.click(screen.getAllByRole('button', { name: /Иван/ })[0]);
    expect(context.openChat).toHaveBeenCalledWith('Иван');

    // The contact list row is a separate entry point to the same chat.
    fireEvent.click(document.querySelector('.chat-contact'));
    expect(context.openChat).toHaveBeenLastCalledWith('Анна');

    fireEvent.click(screen.getByRole('button', { name: 'Настройки' }));
    expect(
      screen.getByRole('dialog', { name: 'Настройки чата' })
    ).toBeVisible();
    fireEvent.click(screen.getByLabelText('Закрыть'));
  });

  it('detail view goes back, opens the friend profile and the settings', () => {
    const context = makeContext({
      state: {
        guest: false,
        chatConsent: true,
        selectedChat: 'Анна',
        chatContacts: [{ name: 'Анна', avatar: '/anna.jpg' }],
      },
    });
    useApp.mockReturnValue(context);
    render(<Chat />);

    fireEvent.click(screen.getByLabelText('Назад'));
    expect(context.go).toHaveBeenCalledWith('chat');

    fireEvent.click(screen.getByRole('button', { name: /Анна/ }));
    expect(context.openFriendProfile).toHaveBeenCalledWith('Анна');

    fireEvent.click(screen.getByLabelText('Настройки чата'));
    expect(
      screen.getByRole('dialog', { name: 'Настройки чата' })
    ).toBeVisible();
  });

  it('asks for the chat settings when consent is missing', () => {
    const context = makeContext({
      state: {
        guest: false,
        chatConsent: false,
        selectedChat: 'Анна',
        chatContacts: [{ name: 'Анна', avatar: '/anna.jpg' }],
      },
      sendChat: vi.fn(),
    });
    useApp.mockReturnValue(context);
    render(<Chat />);

    const input = screen.getByLabelText('Сообщение');
    fireEvent.change(input, { target: { value: 'Привет' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(context.sendChat).not.toHaveBeenCalled();
    expect(
      screen.getByRole('dialog', { name: 'Настройки чата' })
    ).toBeVisible();
  });
});
