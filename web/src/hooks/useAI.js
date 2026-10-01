import { useCallback } from 'react';
import * as api from '../services/api';

export function useAI({ state, update }) {
  const askAI = useCallback(
    (_text) => {
      update({ screen: 'ai' });
    },
    [update]
  );

  const sendAI = useCallback(async () => {
    const input = document.getElementById('aiInput');
    const text = input?.value.trim();
    if (!text) return;
    const userMsg = { id: `${Date.now()}-u`, type: 'user', text };
    const newMessages = [...state.messages, userMsg];
    update({ messages: newMessages });
    if (input) input.value = '';
    try {
      const data = await api.sendAIMessage(text);
      if (data) {
        const aiMsg = {
          id: `${Date.now()}-a`,
          type: 'ai',
          text: data.answer || data.message || 'Ответ получен',
          classification: data.classification || null,
          plan: data.plan || null,
          diet: data.diet || null,
          timestamp: data.timestamp,
        };
        update({
          messages: [...newMessages, aiMsg],
          aiClassification: data.classification || null,
          aiPlan: data.plan || null,
          aiDiet: data.diet || null,
        });
      }
    } catch {
      update({
        messages: [
          ...newMessages,
          {
            id: `${Date.now()}-e`,
            type: 'ai',
            text: 'Ошибка подключения к AI сервису',
          },
        ],
      });
    }
  }, [state.messages, update]);

  const quickAI = useCallback(
    async (text) => {
      const newMessages = [
        ...state.messages,
        { id: `${Date.now()}-u`, type: 'user', text },
      ];
      update({ messages: newMessages });
      try {
        const data = await api.quickAI(text);
        if (data) {
          const aiMsg = {
            id: `${Date.now()}-a`,
            type: 'ai',
            text: data.answer || data.message || 'Ответ получен',
            classification: data.classification || null,
            plan: data.plan || null,
            diet: data.diet || null,
            timestamp: data.timestamp,
          };
          update({
            messages: [...newMessages, aiMsg],
            aiClassification: data.classification || null,
            aiPlan: data.plan || null,
            aiDiet: data.diet || null,
          });
        }
      } catch {
        update({
          messages: [
            ...newMessages,
            {
              id: `${Date.now()}-e`,
              type: 'ai',
              text: 'Ошибка подключения к AI сервису',
            },
          ],
        });
      }
    },
    [state.messages, update]
  );

  const askAIForDay = useCallback(
    (day) => {
      update({ screen: 'ai', selectedCalendarDay: day });
    },
    [update]
  );

  return {
    askAI,
    sendAI,
    quickAI,
    askAIForDay,
  };
}
