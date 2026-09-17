import { useApp } from '../contexts/AppContext';

export default function AI() {
  const { state, t, sendAI, setAI } = useApp();
  const history = state.aiHistory || [];
  const text = state.aiText || '';

  return (
    <section className='ai'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('AI-советник', 'AI Advisor')}</div>
        </div>
        <div className='panel-body'>
          <div className='chat'>
            {history.map((m, i) => (
              <div
                key={i}
                className={`msg ${m.role === 'user' ? 'user' : 'bot'}`}
              >
                <div>{m.text}</div>
              </div>
            ))}
          </div>
          <div className='field'>
            <label>{t('Запрос', 'Prompt')}</label>
            <input
              id='aiPrompt'
              value={text}
              onChange={(e) => setAI(e.target.value)}
              placeholder={t('Например: план на неделю', 'e.g. weekly plan')}
            />
          </div>
          <button className='primary full' onClick={sendAI}>
            {t('Отправить', 'Send')}
          </button>
        </div>
      </div>
    </section>
  );
}
