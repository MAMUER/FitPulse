import Modal from './Modal';

export default function ChatSettingsModal({ state, t, onClose, onSave }) {
  const guestNote = state.guest
    ? t(
        'Для гостя личные чаты отключены. Можно настроить локальные параметры приватности.',
        'Private chats are disabled for guests. You can configure local privacy settings.'
      )
    : t(
        'Управляйте видимостью и уведомлениями чата.',
        'Control chat visibility and notifications.'
      );

  return (
    <Modal
      className='fp-profile-modal'
      label={t('Настройки чата', 'Chat settings')}
      onClose={onClose}
    >
      <section className='fp-profile-box'>
        <div className='between'>
          <h2>{t('Настройки чата', 'Chat settings')}</h2>
          <button
            type='button'
            className='icon'
            aria-label={t('Закрыть', 'Close')}
            onClick={onClose}
          >
            ×
          </button>
        </div>
        <p className='muted' style={{ fontSize: 10 }}>
          {guestNote}
        </p>
        <div className='guest-banner'>
          <strong>
            {t('Региональный режим: РФ', 'Regional mode: Russia')}
          </strong>
          <span className='muted'>
            {t(
              'Для публикации в РФ оператору нужно отдельно проверить требования к персональным данным, хранению и модерации.',
              'For publication in Russia, the operator must separately verify personal-data, storage and moderation requirements.'
            )}
          </span>
        </div>
        <label className='setting'>
          <span>{t('Показывать онлайн-статус', 'Show online status')}</span>
          <input
            type='checkbox'
            id='csOnline'
            defaultChecked={state.chatSettings?.onlineStatus}
          />
        </label>
        <label className='setting'>
          <span>{t('Отметки о прочтении', 'Read receipts')}</span>
          <input
            type='checkbox'
            id='csRead'
            defaultChecked={state.chatSettings?.readReceipts}
          />
        </label>
        <label className='setting'>
          <span>{t('Уведомления', 'Notifications')}</span>
          <input
            type='checkbox'
            id='csNotify'
            defaultChecked={state.chatSettings?.notifications}
          />
        </label>
        <label className='consent-row'>
          <input
            id='csConsent'
            type='checkbox'
            defaultChecked={state.chatConsent}
          />
          <span>
            {t(
              'Я согласен(на) на обработку данных, необходимых для чата, и ознакомлен(а) с Политикой конфиденциальности.',
              'I consent to processing necessary for chat and have read the Privacy Policy.'
            )}
          </span>
        </label>
        <button type='button' className='primary full' onClick={onSave}>
          {t('Сохранить', 'Save')}
        </button>
      </section>
    </Modal>
  );
}
