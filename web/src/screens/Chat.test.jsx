import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Chat from './Chat';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'chat',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    chatContacts: [{ name: 'Анна', last: 'Привет!', time: '12:30', unread: 1 }],
    chatMessages: [],
    selectedChat: null,
  },
  t: (ru, _en) => ru,
  sendChat: vi.fn(),
  chatSettings: vi.fn(),
  go: vi.fn(),
};

function renderChat() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <AppProvider>
      <Chat />
    </AppProvider>
  );
}

describe('Chat', () => {
  it('renders chat screen', () => {
    renderChat();
    expect(screen.getByText('Чат с поддержкой')).toBeInTheDocument();
  });
});
