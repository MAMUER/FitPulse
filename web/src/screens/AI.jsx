import { useEffect, useRef, useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Ai() {
  const { state, t, sendAI, quickAI, clearChat } = useApp();
  const [input, setInput] = useState('');
  const messagesRef = useRef(null);
  const messages = state.messages || [];
  const classification = state.aiClassification;
  const plan = state.aiPlan;
  const diet = state.aiDiet;

  // biome-ignore lint/correctness/useExhaustiveDependencies: scroll follows new messages
  useEffect(() => {
    /* v8 ignore next 3 */
    if (messagesRef.current) {
      messagesRef.current.scrollTop = messagesRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSend = async () => {
    const text = input.trim();
    if (!text) return;
    setInput('');
    await sendAI(text);
  };

  const quickQuestions = [
    ['Меню', 'Составь меню на сегодня', 'Make a menu for today'],
    [
      'Тренировки · неделя',
      'Составь план тренировок на неделю',
      'Make a workout plan for the week',
    ],
    [
      'Тренировки · месяц',
      'Составь план тренировок на месяц',
      'Make a workout plan for the month',
    ],
    [
      'Тело',
      'Как улучшить композицию тела?',
      'How to improve body composition?',
    ],
  ];

  const renderBlock = (data, title) => {
    if (!data) return null;
    return (
      <div className='ai-block'>
        <div className='ai-block-title'>{title}</div>
        <pre className='ai-block-pre'>{JSON.stringify(data, null, 2)}</pre>
      </div>
    );
  };

  return (
    <section className='ai'>
      <div className='ai-head'>
        <div className='ai-avatar'>
          <i className='fas fa-robot' />
        </div>
        <div>
          <h2>FitPulse AI</h2>
          <span style={{ color: 'var(--blue)', fontSize: 9 }}>
            {t(
              'Тренер · нутрициолог · планировщик',
              'Trainer · nutritionist · planner'
            )}
          </span>
        </div>
        {messages.length > 0 && (
          <button
            type='button'
            className='secondary'
            style={{ marginLeft: 'auto' }}
            onClick={clearChat}
          >
            <i className='fas fa-trash' /> {t('Очистить чат', 'Clear chat')}
          </button>
        )}
      </div>

      <section className='card' style={{ marginTop: 17 }}>
        <div className='eyebrow'>{t('УМНЫЙ ПОМОЩНИК', 'SMART ASSISTANT')}</div>
        <h3 style={{ marginTop: 8 }}>
          {t('Что планируем?', 'What are we planning?')}
        </h3>
        <p style={{ marginTop: 5, color: 'var(--muted)', fontSize: 10 }}>
          {t(
            'Напиши запрос обычными словами — я подберу подходящий вариант.',
            "Write your request in plain words — I'll find the best option."
          )}
        </p>
        <div className='quick-questions'>
          {quickQuestions.map(([label, ru, en]) => (
            <button
              type='button'
              key={label}
              onClick={() => {
                quickAI(t(ru, en));
              }}
            >
              {label}
            </button>
          ))}
        </div>
      </section>

      <div className='messages' ref={messagesRef}>
        {messages.map((message, index) => (
          <div className={`message ${message.type}`} key={message.id || index}>
            {message.text}
          </div>
        ))}
      </div>

      {renderBlock(classification, t('Классификация', 'Classification'))}
      {renderBlock(plan, t('План тренировок', 'Training plan'))}
      {renderBlock(diet, t('План питания', 'Nutrition plan'))}

      <div className='chat'>
        <input
          id='aiInput'
          value={input}
          onChange={(event) => setInput(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault();
              void handleSend();
            }
          }}
          placeholder={t(
            'Например: у меня мало времени',
            "E.g. I don't have much time"
          )}
        />
        <button
          type='button'
          aria-label={t('Отправить', 'Send')}
          onClick={handleSend}
        >
          <i className='fas fa-arrow-up' />
        </button>
      </div>
    </section>
  );
}
