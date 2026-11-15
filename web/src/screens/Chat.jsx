import { useState } from 'react';
import ChatSettingsModal from '../components/ChatSettingsModal';
import { useApp } from '../contexts/AppContext';

function getStatusClass(status) {
  if (status === 'gym' || status === 'pool' || status === 'dnd') {
    return `online-${status}`;
  }
  return '';
}

export default function Chat() {
  const {
    state,
    t,
    go,
    openLegal,
    openChat,
    openFriendProfile,
    saveChatSettings,
    sendChat,
  } = useApp();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [text, setText] = useState('');
  const contacts = state.chatContacts || [];
  const selectedContact = contacts.find(
    (contact) => contact.name === state.selectedChat
  );
  const isDetail = Boolean(state.selectedChat);

  const handleSend = () => {
    if (!text.trim()) return;
    if (!state.chatConsent) {
      setSettingsOpen(true);
      return;
    }
    sendChat();
    setText('');
  };

  const handleSaveSettings = () => {
    saveChatSettings();
    setSettingsOpen(false);
  };

  const guestView = (
    <>
      <div className='page-title'>
        <div className='eyebrow'>MESSENGER</div>
        <h2>{t('Чат', 'Chat')}</h2>
        <p>
          {t(
            'Гостевой режим с ограниченными функциями приватности.',
            'Guest mode with limited privacy features.'
          )}
        </p>
      </div>
      <div className='chat-locked'>
        <i
          className='fas fa-lock'
          style={{ fontSize: 30, color: 'var(--blue)' }}
        />
        <h3 style={{ marginTop: 10 }}>
          {t(
            'Личные сообщения недоступны гостю',
            'Private messaging is unavailable to guests'
          )}
        </h3>
        <p className='muted' style={{ margin: '8px 0' }}>
          {t(
            'Для реального общения войдите или зарегистрируйтесь. До этого можно настроить локальные параметры чата.',
            'Sign in or register for real messaging. Until then, you can configure local chat settings.'
          )}
        </p>
        <button
          type='button'
          className='secondary full'
          onClick={() => setSettingsOpen(true)}
        >
          {t('Настройки чата', 'Chat settings')}
        </button>
        <button
          type='button'
          className='primary full'
          style={{ marginTop: 7 }}
          onClick={() => go('login')}
        >
          {t('Войти', 'Sign in')}
        </button>
        <button
          type='button'
          className='secondary full'
          style={{ marginTop: 7 }}
          onClick={() => go('register')}
        >
          {t('Зарегистрироваться', 'Register')}
        </button>
        <div className='auth-link'>
          <button
            type='button'
            className='legal-link'
            onClick={() => openLegal('privacy')}
          >
            {t('Политика конфиденциальности', 'Privacy Policy')}
          </button>
          {' · '}
          <button
            type='button'
            className='legal-link'
            onClick={() => openLegal('terms')}
          >
            {t('Соглашение', 'Terms')}
          </button>
          {' · '}
          <button
            type='button'
            className='legal-link'
            onClick={() => openLegal('consent')}
          >
            {t('Персональные данные', 'Personal Data')}
          </button>
        </div>
      </div>
    </>
  );

  const detailView = (
    <>
      <div className='page-title chat-detail-title'>
        <button
          type='button'
          className='icon'
          aria-label={t('Назад', 'Back')}
          onClick={() => go('chat')}
        >
          <i className='fas fa-arrow-left' />
        </button>
        <button
          type='button'
          className='chat-contact-heading'
          onClick={() => openFriendProfile(selectedContact?.name)}
        >
          <img src={selectedContact?.avatar} alt='' />
          <span>
            <small className='eyebrow'>{t('Чат', 'Chat')}</small>
            <strong>{state.selectedChat}</strong>
          </span>
        </button>
        <button
          type='button'
          className='icon'
          aria-label={t('Настройки чата', 'Chat settings')}
          onClick={() => setSettingsOpen(true)}
        >
          <i className='fas fa-gear' />
        </button>
      </div>
      <div className='chat-messages'>
        {(state.chatMessages || []).map((message, index) => (
          <div
            key={`${message.time || ''}-${index}`}
            className={`chat-msg ${message.sent ? 'sent' : 'received'}`}
          >
            {message.text}
            <span className='time'>{message.time}</span>
          </div>
        ))}
      </div>
      <div className='chat'>
        <input
          id='chatInput'
          aria-label={t('Сообщение', 'Message')}
          placeholder={t('Введите сообщение...', 'Type a message...')}
          value={text}
          onChange={(event) => setText(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') handleSend();
          }}
        />
        <button
          type='button'
          aria-label={t('Отправить', 'Send')}
          onClick={handleSend}
        >
          <i className='fas fa-arrow-up' />
        </button>
      </div>
      <div className='auth-link' style={{ marginTop: 8 }}>
        {t(
          'Стикеры отключены в базовой версии чата; используется текстовый чат.',
          'Stickers are disabled in the base chat; text chat is used.'
        )}
      </div>
    </>
  );

  const contactsView = (
    <>
      <div className='page-title'>
        <div className='eyebrow'>MESSENGER</div>
        <h2>{t('Чат', 'Chat')}</h2>
        <p>
          {t(
            'Общайся с другими пользователями FitPulse',
            'Chat with other FitPulse users'
          )}
        </p>
      </div>
      <div className='between'>
        <div className='section'>
          <h3>
            {t('В сети', 'Online')} · {state.onlineUsers || 0}
          </h3>
        </div>
        <button
          type='button'
          className='secondary'
          onClick={() => setSettingsOpen(true)}
        >
          {t('Настройки', 'Settings')}
        </button>
      </div>
      <div className='fp-online-grid'>
        {contacts
          .filter((contact) => contact.online)
          .map((contact) => (
            <button
              type='button'
              key={contact.name}
              className={`fp-online-card ${getStatusClass(contact.status)}`}
              onClick={() => openChat(contact.name)}
            >
              <img src={contact.avatar} alt='' />
              <strong>{contact.name}</strong>
              <small>{contact.status}</small>
            </button>
          ))}
      </div>
      <div className='chat-list'>
        {contacts.map((contact) => (
          <button
            type='button'
            key={contact.name}
            className='chat-contact'
            onClick={() => openChat(contact.name)}
          >
            <span className='avatar'>
              <img src={contact.avatar} alt='' />
            </span>
            <span className='info'>
              <strong>{contact.name}</strong>
              <small>{contact.last}</small>
            </span>
            <span className='time'>{contact.time}</span>
            {contact.unread > 0 && (
              <span className='unread'>{contact.unread}</span>
            )}
          </button>
        ))}
      </div>
    </>
  );

  let chatView = contactsView;
  if (state.guest) {
    chatView = guestView;
  } else if (isDetail) {
    chatView = detailView;
  }

  return (
    <section className='chat-screen'>
      {chatView}
      {settingsOpen && (
        <ChatSettingsModal
          state={state}
          t={t}
          onClose={() => setSettingsOpen(false)}
          onSave={handleSaveSettings}
        />
      )}
    </section>
  );
}
