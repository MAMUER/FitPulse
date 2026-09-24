import { useApp } from '../contexts/AppContext';

export default function Chat() {
  const { state, t, sendChat, setChat } = useApp();
  const history = state.chatHistory || [];
  const text = state.chatText || '';

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
              <div
                key={m.id || i}
                className={`msg ${m.role === 'user' ? 'user' : 'bot'}`}
              >
                <div>{m.text}</div>
              </div>
            ))}
          </div>
          <div className='field'>
            <label>{t('Сообщение', 'Message')}</label>
            <input
              id='chatInput'
              value={text}
              onChange={(e) => setChat(e.target.value)}
              placeholder={t('Напишите сообщение...', 'Write a message...')}
            />
          </div>
          <button className='primary full' onClick={sendChat}>
            {t('Отправить', 'Send')}
          </button>
        </div>
      </div>
    </section>
  );
}
