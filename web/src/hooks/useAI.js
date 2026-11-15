import { useCallback } from 'react';
import * as api from '../services/api';

export function useAI({ state, update }) {
  const askAI = useCallback(
    (_text) => {
      update({ screen: 'ai' });
    },
    [update]
  );

  const askAIForDay = useCallback(
    (day) => {
      update({ screen: 'ai', selectedCalendarDay: day });
    },
    [update]
  );

  const askAIRequest = useCallback(
    async (text, sendRequest, userMsg) => {
      const newMessages = [...state.messages, userMsg];
      update({ messages: newMessages });

      try {
        const data = await sendRequest(text);

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

  const sendAI = useCallback(
    async (text) => {
      const input = document.getElementById('aiInput');
      const value = (
        typeof text === 'string' ? text : input?.value || ''
      ).trim();
      if (!value) return;

      if (input) input.value = '';
      await askAIRequest(value, api.sendAIMessage, {
        id: `${Date.now()}-u`,
        type: 'user',
        text: value,
      });
    },
    [askAIRequest]
  );

  const quickAI = useCallback(
    async (text) => {
      await askAIRequest(text, api.quickAI, {
        id: `${Date.now()}-u`,
        type: 'user',
        text,
      });
    },
    [askAIRequest]
  );

  return {
    askAI,
    sendAI,
    quickAI,
    askAIForDay,
  };
}
