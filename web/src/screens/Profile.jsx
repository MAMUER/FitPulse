import { useEffect, useState } from 'react';
import ChatSettingsModal from '../components/ChatSettingsModal';
import Modal from '../components/Modal';
import { useApp } from '../contexts/AppContext';

function friendCardClass(friend) {
  const statusClass = friend.online ? `online-${friend.status}` : '';
  return `friend-card ${statusClass}`.trim();
}

export default function Profile() {
  const {
    state,
    t,
    go,
    openLegal,
    toggleLanguage,
    toggleTheme,
    toggleHighContrast,
    load2FAStatus,
    setup2FA,
    disable2FA,
    loadProfile,
    loadFriends,
    loadAchievements,
    changePassword,
    deleteAccount,
    uploadPhoto,
    editProfile,
    showDevices,
    showRestrictions,
    nutritionSettings,
    saveChatSettings: persistChatSettings,
    addFriend,
    openFriendProfile,
    showAchievement,
  } = useApp();
  const [code, setCode] = useState('');
  const [loading, setLoading] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [friendModalOpen, setFriendModalOpen] = useState(false);
  const [friendName, setFriendName] = useState('');
  const [chatSettingsOpen, setChatSettingsOpen] = useState(false);

  useEffect(() => {
    loadProfile();
    loadFriends();
    load2FAStatus();
    void loadAchievements();
  }, [loadProfile, loadFriends, load2FAStatus, loadAchievements]);

  const name = state.profile?.name || state.profile?.email?.split('@')[0] || '';
  const friends = state.friends || [];
  const achievements =
    state.achievementsBackend && state.achievementsBackend.length > 0
      ? state.achievementsBackend
      : state.achievements || [];
  const eventCount =
    state.events?.length ||
    Object.values(state.calendarEvents || {}).reduce(
      (count, events) => count + (Array.isArray(events) ? events.length : 0),
      0
    );
  const completedAchievements = achievements.filter((item) => item.done).length;
  const level = state.level || 1;
  const points = state.points || 0;
  const streak = state.streak || 0;
  const maxStreak = state.maxStreak || 0;
  const pct = Math.min(100, (points % 1000) / 10);
  const friendStatus = (status) => {
    const labels = {
      gym: t('В зале', 'At gym'),
      pool: t('В бассейне', 'At pool'),
      dnd: t('Не беспокоить', 'Do not disturb'),
      offline: t('Не в сети', 'Offline'),
    };
    return labels[status] || t('В сети', 'Online');
  };

  const handleDisable2FA = async () => {
    setLoading(true);
    await disable2FA(code);
    setCode('');
    setLoading(false);
  };

  const handleDeleteAccount = () => {
    if (deleteAccount()) go('login');
  };

  const handleSavePassword = () => {
    changePassword();
    setPasswordOpen(false);
  };

  const handleSaveChatSettings = () => {
    persistChatSettings();
    setChatSettingsOpen(false);
  };

  return (
    <section className='profile'>
      <div className='page-title'>
        <div className='eyebrow'>FITPULSE IDENTITY</div>
        <h2>{t('Профиль', 'Profile')}</h2>
        <p>
          {t(
            'Фото, цели, достижения и настройки приложения.',
            'Photo, goals, achievements and app settings.'
          )}
        </p>
      </div>

      <div className='profile-card'>
        <div className='profile-header'>
          <div className='profile-photo-large'>
            {state.profileImage ? (
              <img
                src={state.profileImage}
                alt={t('Фото профиля', 'Profile photo')}
              />
            ) : (
              name[0]
            )}
          </div>
          <div className='profile-info'>
            <h2>{name}</h2>
            <div className='status-text'>
              {state.profile?.status || t('Активен', 'Active')} ·{' '}
              {state.activeDays || 0} {t('активных дней', 'active days')}
            </div>
            <label className='upload-btn'>
              <i className='fas fa-camera' />{' '}
              {t('Изменить фото', 'Change photo')}
              <input type='file' accept='image/*' onChange={uploadPhoto} />
            </label>
          </div>
        </div>
        <div className='profile-stats'>
          <div>
            <b>{state.activeDays || 0}</b>
            <small>{t('Серия', 'Streak')}</small>
          </div>
          <div>
            <b>{eventCount}</b>
            <small>{t('Событий', 'Events')}</small>
          </div>
          <div>
            <b>
              {completedAchievements}/{achievements.length}
            </b>
            <small>{t('Достижений', 'Achievements')}</small>
          </div>
          <button
            type='button'
            className='profile-stat'
            onClick={() => go('chat')}
            aria-label={t('Друзья', 'Friends')}
          >
            <b>
              {state.profile?.friends ||
                friends.filter((friend) => friend.added).length}
            </b>
            <small>{t('Друзей', 'Friends')}</small>
          </button>
        </div>
        <div className='profile-actions'>
          <button type='button' className='secondary' onClick={editProfile}>
            <i className='fas fa-edit' /> {t('Редактировать', 'Edit')}
          </button>
          <button
            type='button'
            className='secondary'
            onClick={() => go('chat')}
          >
            <i className='fas fa-comments' /> {t('Чат', 'Chat')}
          </button>
        </div>
      </div>

      <div className='fp-level-card'>
        <div className='fp-level-title'>
          <strong>
            ⭐ {t('Уровень', 'Level')} {level}
          </strong>
          <span>
            {points} / {level * 1000} XP
          </span>
        </div>
        <p style={{ marginTop: 7, color: 'var(--muted)', fontSize: 10 }}>
          {t(
            'Продолжай серию, чтобы открыть новый уровень.',
            'Keep your streak to unlock the next level.'
          )}
        </p>
        <div className='fp-xp'>
          <i style={{ width: `${pct}%` }} />
        </div>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            marginTop: 6,
            fontSize: 8,
            color: 'var(--muted)',
          }}
        >
          <span>
            🔥 {t('Серия', 'Streak')}: {streak} {t('дней', 'days')}
          </span>
          <span>
            🏆 {t('Макс. серия', 'Max streak')}: {maxStreak}
          </span>
        </div>
      </div>
      <div className='fp-trophy-grid'>
        <div className='fp-trophy'>
          <i className='fas fa-trophy' />
          <strong>{level}</strong>
          <small>{t('уровень', 'level')}</small>
        </div>
        <div className='fp-trophy'>
          <i className='fas fa-medal' />
          <strong>{completedAchievements}</strong>
          <small>{t('наград', 'awards')}</small>
        </div>
        <div className='fp-trophy'>
          <i className='fas fa-star' />
          <strong>{streak}</strong>
          <small>{t('серия', 'streak')}</small>
        </div>
      </div>

      <div className='section'>
        <h3>
          {t('Достижения', 'Achievements')}{' '}
          <span style={{ color: 'var(--gold)', fontSize: 12 }}>
            ⭐ {t('Золото', 'Gold')}
          </span>
        </h3>
      </div>
      <div className='grid-2'>
        {achievements.map((achievement) => (
          <button
            type='button'
            key={achievement.name}
            className={`fp-achievement ${achievement.done ? 'done' : ''}`}
            onClick={() => showAchievement(achievement.name)}
          >
            <i className={`fas ${achievement.icon}`} />
            <strong>{achievement.name}</strong>
            <small>
              {achievement.done
                ? t('Выполнено ✓', 'Done ✓')
                : t('В процессе', 'In progress')}
            </small>
          </button>
        ))}
      </div>

      <div className='section'>
        <div className='between'>
          <h3>{t('Друзья', 'Friends')}</h3>
          <button
            type='button'
            className='secondary'
            onClick={() => setFriendModalOpen(true)}
          >
            <i className='fas fa-user-plus' /> {t('Добавить', 'Add')}
          </button>
        </div>
        <p className='muted fs-10' style={{ marginTop: 4 }}>
          {t('Онлайн:', 'Online:')} {state.onlineUsers || 0}
        </p>
      </div>
      <div className='friends-grid'>
        {friends.map((friend) => (
          <button
            type='button'
            key={friend.id || friend.name}
            className={friendCardClass(friend)}
            onClick={() => openFriendProfile(friend)}
          >
            <span className='avatar'>
              {friend.image ? (
                <img src={friend.image} alt='' />
              ) : (
                friend.name?.[0]
              )}
            </span>
            <span className='name'>{friend.name}</span>
            <span className='status-text'>
              <span
                className={`status-dot ${friend.online ? friend.status : 'offline'}`}
              />
              {friend.online
                ? friendStatus(friend.status)
                : t('Не в сети', 'Offline')}
            </span>
          </button>
        ))}
      </div>

      {state.surveyDeferred && !state.surveyCompleted && (
        <div className='guest-banner'>
          <strong>{t('Опрос отложен', 'Survey postponed')}</strong>
          <span className='muted'>
            {t(
              'Заполните его позже, чтобы FitPulse учитывал цели, ограничения и предпочтения.',
              'Complete it later so FitPulse can use your goals, restrictions and preferences.'
            )}
          </span>
          <button
            type='button'
            className='primary full'
            style={{ marginTop: 8 }}
            onClick={() => go('survey')}
          >
            {t('Продолжить опрос', 'Continue survey')}
          </button>
        </div>
      )}

      <div className='section'>
        <h3>{t('Настройки', 'Settings')}</h3>
      </div>
      <div className='settings'>
        <button type='button' className='setting' onClick={toggleTheme}>
          <i className='fas fa-circle-half-stroke' />
          <span>
            {t('Тема', 'Theme')}
            <small>
              {state.theme === 'dark'
                ? t('Тёмная', 'Dark')
                : t('Светлая', 'Light')}
            </small>
          </span>
          <i className='fas fa-chevron-right' />
        </button>
        <button type='button' className='setting' onClick={toggleLanguage}>
          <i className='fas fa-globe' />
          <span>
            {t('Язык', 'Language')}
            <small>{state.language === 'ru' ? 'Русский' : 'English'}</small>
          </span>
          <i className='fas fa-chevron-right' />
        </button>
        <button type='button' className='setting' onClick={toggleHighContrast}>
          <i className='fas fa-eye' />
          <span>
            {t('Режим высокой контрастности', 'High contrast mode')}
            <small>
              {state.highContrast ? t('Вкл', 'On') : t('Выкл', 'Off')}
            </small>
          </span>
          <i className='fas fa-chevron-right' />
        </button>
        <button type='button' className='setting' onClick={nutritionSettings}>
          <i className='fas fa-utensils' />
          <span>
            {t('Питание', 'Nutrition')}
            <small>
              {t(
                'Аллергии, ограничения, цели',
                'Allergies, restrictions, goals'
              )}
            </small>
          </span>
          <i className='fas fa-chevron-right' />
        </button>
        <button type='button' className='setting' onClick={editProfile}>
          <i className='fas fa-user-edit' />
          <span>
            {t('Редактировать профиль', 'Edit profile')}
            <small>
              {t('Имя, пол, email, телефон', 'Name, gender, email, phone')}
            </small>
          </span>
          <i className='fas fa-chevron-right' />
        </button>
        <div className='setting'>
          <i className='fas fa-shield-alt' />
          <span>
            {t('Двухфакторная аутентификация', 'Two-factor authentication')}
            <small>
              {state.profile?.twoFactor ? t('Вкл', 'On') : t('Выкл', 'Off')}
            </small>
          </span>
          {state.profile?.twoFactor ? (
            <button
              type='button'
              className='danger'
              onClick={handleDisable2FA}
              disabled={loading}
            >
              {loading
                ? t('Отключение...', 'Disabling...')
                : t('Отключить', 'Disable')}
            </button>
          ) : (
            <button type='button' className='primary' onClick={setup2FA}>
              {t('Включить', 'Enable')}
            </button>
          )}
        </div>
        {state.profile?.twoFactor && (
          <label className='field'>
            <span>{t('Код для отключения', 'Code to disable')}</span>
            <input
              value={code}
              onChange={(event) => setCode(event.target.value)}
              maxLength={6}
            />
          </label>
        )}
        <button
          type='button'
          className='setting'
          onClick={() => setPasswordOpen(true)}
        >
          <i className='fas fa-key' />
          <span>
            {t('Сменить пароль', 'Change password')}
            <small>{t('Обновить пароль', 'Update password')}</small>
          </span>
          <i className='fas fa-chevron-right' />
        </button>
        <button type='button' className='setting' onClick={showDevices}>
          <i className='fas fa-plug' />
          <span>
            {t('Подключенные устройства', 'Connected devices')}
            <small>{(state.profile?.devices || []).join(', ')}</small>
          </span>
          <i className='fas fa-chevron-right' />
        </button>
        <button type='button' className='setting' onClick={showRestrictions}>
          <i className='fas fa-ban' />
          <span>
            {t('Ограничения', 'Restrictions')}
            <small>{t('Аллергии и здоровье', 'Allergies & health')}</small>
          </span>
          <i className='fas fa-chevron-right' />
        </button>
      </div>

      <button
        type='button'
        className='setting'
        onClick={() => setChatSettingsOpen(true)}
      >
        <i className='fas fa-comments' />
        <span>
          {t('Настройки чата', 'Chat settings')}
          <small>
            {t(
              'Приватность, статус, уведомления',
              'Privacy, status, notifications'
            )}
          </small>
        </span>
        <i className='fas fa-chevron-right' />
      </button>

      <div className='danger-zone'>
        <strong style={{ color: 'var(--red)' }}>
          {t('Опасная зона', 'Danger zone')}
        </strong>
        <p className='muted' style={{ fontSize: 9, margin: '5px 0 9px' }}>
          {t(
            'Удаление аккаунта очищает локальные данные демо-версии. Серверное удаление требует backend.',
            'Account deletion clears local demo data. Server-side deletion requires a backend.'
          )}
        </p>
        <button
          type='button'
          className='secondary full'
          onClick={() => setPasswordOpen(true)}
        >
          <i className='fas fa-key' /> {t('Изменить пароль', 'Change password')}
        </button>
        <button
          type='button'
          className='secondary full'
          style={{
            marginTop: 7,
            borderColor: 'var(--red)',
            color: 'var(--red)',
          }}
          onClick={handleDeleteAccount}
        >
          <i className='fas fa-trash' />{' '}
          {t('Удалить аккаунт', 'Delete account')}
        </button>
      </div>

      <div className='section'>
        <h3>{t('Документы и данные', 'Documents & data')}</h3>
      </div>
      <div className='form-row'>
        <button
          type='button'
          className='secondary'
          onClick={() => openLegal('privacy')}
        >
          {t('Конфиденциальность', 'Privacy')}
        </button>
        <button
          type='button'
          className='secondary'
          onClick={() => openLegal('terms')}
        >
          {t('Соглашение', 'Terms')}
        </button>
        <button
          type='button'
          className='secondary'
          onClick={() => openLegal('consent')}
        >
          {t('Персональные данные', 'Personal Data')}
        </button>
      </div>

      {passwordOpen && (
        <Modal
          className='fp-profile-modal'
          label={t('Смена пароля', 'Change password')}
          onClose={() => setPasswordOpen(false)}
        >
          <section className='fp-profile-box'>
            <div className='between'>
              <h2>{t('Смена пароля', 'Change password')}</h2>
              <button
                type='button'
                className='icon'
                aria-label={t('Закрыть', 'Close')}
                onClick={() => setPasswordOpen(false)}
              >
                ×
              </button>
            </div>
            <label className='form-label'>
              {t('Текущий пароль', 'Current password')}
              <input id='oldPass' type='password' />
            </label>
            <label className='form-label'>
              {t('Новый пароль', 'New password')}
              <input id='newPass' type='password' />
            </label>
            <label className='form-label'>
              {t('Подтвердите новый пароль', 'Confirm new password')}
              <input id='newPass2' type='password' />
            </label>
            <button
              type='button'
              className='primary full'
              onClick={handleSavePassword}
            >
              {t('Сохранить', 'Save')}
            </button>
          </section>
        </Modal>
      )}
      {chatSettingsOpen && (
        <ChatSettingsModal
          state={state}
          t={t}
          onClose={() => setChatSettingsOpen(false)}
          onSave={handleSaveChatSettings}
        />
      )}

      {friendModalOpen && (
        <Modal
          className='fp-profile-modal'
          label={t('Добавить друга', 'Add friend')}
          onClose={() => setFriendModalOpen(false)}
        >
          <section className='fp-profile-box'>
            <div className='field'>
              <label htmlFor='friendName'>
                {t('Имя друга', 'Friend name')}
              </label>
              <input
                id='friendName'
                type='text'
                value={friendName}
                placeholder={t('Например, Анна', 'e.g. Anna')}
                onChange={(e) => setFriendName(e.target.value)}
              />
            </div>
            <button
              type='button'
              className='primary full'
              style={{ marginTop: 12 }}
              onClick={async () => {
                const ok = await addFriend(friendName);
                if (ok) {
                  setFriendName('');
                  setFriendModalOpen(false);
                }
              }}
            >
              <i className='fas fa-user-plus' /> {t('Добавить', 'Add')}
            </button>
          </section>
        </Modal>
      )}
    </section>
  );
}
