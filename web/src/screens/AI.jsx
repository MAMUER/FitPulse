import { useEffect, useRef, useState } from 'react';
import Panel from '../components/Panel';
import { useApp } from '../contexts/AppContext';

export default function Ai() {
  const { state, t, sendAI, go } = useApp();
  const [input, setInput] = useState('');
  const messages = state.messages || [];
  const classification = state.aiClassification;
  const plan = state.aiPlan;
  const diet = state.aiDiet;
  const chatRef = useRef(null);

  useEffect(() => {
    if (chatRef.current) {
      chatRef.current.scrollTop = chatRef.current.scrollHeight;
    }
  }, []);

  const handleSend = () => {
    if (!input.trim()) return;
    sendAI();
    setInput('');
  };

  const renderJSON = (data, title) => {
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
      <Panel
        title={t('AI-советник', 'AI Advisor')}
        right={
          <button
            type='button'
            className='secondary'
            onClick={() => go('home')}
          >
            {t('Назад', 'Back')}
          </button>
        }
      >
        <div className='panel-body'>
          <div className='chat' ref={chatRef}>
            {messages.map((m) => (
              <div
                key={m.id}
                className={`msg ${m.type === 'user' ? 'user' : 'bot'}`}
              >
                <div>{m.text}</div>
                {m.classification && (
                  <div className='ai-meta'>
                    {renderJSON(
                      m.classification,
                      t('Классификация', 'Classification')
                    )}
                  </div>
                )}
                {m.plan && (
                  <div className='ai-meta'>
                    {renderJSON(m.plan, t('План', 'Plan'))}
                  </div>
                )}
                {m.diet && (
                  <div className='ai-meta'>
                    {renderJSON(m.diet, t('Питание', 'Diet'))}
                  </div>
                )}
              </div>
            ))}
          </div>
          {classification && !messages.some((m) => m.classification) && (
            <div className='ai-summary'>
              {renderJSON(classification, t('Классификация', 'Classification'))}
              {renderJSON(plan, t('План', 'Plan'))}
              {renderJSON(diet, t('Питание', 'Diet'))}
            </div>
          )}
          <div className='field'>
            <label htmlFor='aiInput'>{t('Запрос', 'Prompt')}</label>
            <input
              id='aiInput'
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={t('Например: план на неделю', 'e.g. weekly plan')}
              onKeyDown={(e) => e.key === 'Enter' && handleSend()}
            />
          </div>
          <button type='button' className='primary full' onClick={handleSend}>
            {t('Отправить', 'Send')}
          </button>
        </div>
      </Panel>
    </section>
  );
}
