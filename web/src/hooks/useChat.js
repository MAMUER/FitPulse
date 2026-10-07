import { useCallback } from 'react';
import { t } from '../utils/i18n';

export function useChat({ state, update, notify }) {
  const openChat = useCallback(
    (name) => {
      update({ selectedChat: name, screen: 'chatDetail' });
    },
    [update]
  );

  const sendChat = useCallback(() => {
    if (state.guest) {
      notify(
        t(
          'В гостевом режиме личный чат недоступен.',
          'Private chat is unavailable in guest mode.'
        )
      );
      return;
    }
    const input = document.getElementById('chatInput');
    const text = input?.value.trim();
    if (!text) return;
    const time = new Date().toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
    });
    update({
      chatMessages: [
        ...(state.chatMessages || []),
        { from: 'Я', text, time, sent: true },
      ],
      chatConsent: true,
    });
    if (input) input.value = '';
    setTimeout(() => {
      update({
        chatMessages: [
          ...(state.chatMessages || []),
          {
            from: state.selectedChat || 'Анна',
            text: t(
              'Отлично! Продолжай в том же духе 💪',
              'Great! Keep it up 💪'
            ),
            time: new Date().toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit',
            }),
            sent: false,
          },
        ],
      });
    }, 800);
  }, [state.guest, state.chatMessages, state.selectedChat, update, notify]);

  const saveChatSettings = useCallback(() => {
    const onlineStatus = !!document.getElementById('csOnline')?.checked;
    const readReceipts = !!document.getElementById('csRead')?.checked;
    const notifications = !!document.getElementById('csNotify')?.checked;
    const chatConsent = !!document.getElementById('csConsent')?.checked;
    update({
      chatSettings: {
        ...state.chatSettings,
        onlineStatus,
        readReceipts,
        notifications,
      },
      chatConsent,
    });
  }, [state.chatSettings, update]);

  const addStickerToChat = useCallback((sticker) => {
    const input = document.getElementById('chatInput');
    if (input) {
      input.value += sticker;
      input.focus();
    }
  }, []);

  return {
    openChat,
    sendChat,
    saveChatSettings,
    addStickerToChat,
  };
}
