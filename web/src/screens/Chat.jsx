import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Chat() {
  const { state, t, sendChat } = useApp();
  const [text, setText] = useState('');
  const history = state.chatMessages || [];

  useEffect(() => {
    const input = document.getElementById('chatInput');
    if (input) input.value = text;
  }, [text]);

  const handleSend = () => {
    if (!text.trim()) return;
    sendChat();
    setText('');
  };

  return (
    <section className='chat'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>
            {t('Чат с поддержкой', 'Support chat')}
          </div>
        </div>
        <div className='panel-body'>
          <div className='chat'>
            {history.map((m, i) => (
              <div key={m.id || i} className={`msg ${m.sent ? 'user' : 'bot'}`}>
                <div>{m.text}</div>
              </div>
            ))}
          </div>
          <div className='field'>
            <label htmlFor='chatInput'>{t('Сообщение', 'Message')}</label>
            <input
              id='chatInput'
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={t('Напишите сообщение...', 'Write a message...')}
            />
          </div>
          <button type='button' className='primary full' onClick={handleSend}>
            {t('Отправить', 'Send')}
          </button>
        </div>
      </div>
    </section>
  );
}
