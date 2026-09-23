import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Profile() {
  const {
    state,
    t,
    logout,
    go,
    toggleLanguage,
    toggleTheme,
    toggleHighContrast,
    load2FAStatus,
    setup2FA,
    disable2FA,
    loadProfile,
    saveProfile,
  } = useApp();
  const [code, setCode] = useState('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadProfile();
    load2FAStatus();
  }, [loadProfile, load2FAStatus]);

  const handleDisable = async () => {
    setLoading(true);
    await disable2FA(code);
    setCode('');
    setLoading(false);
  };

  const handleSaveProfile = async () => {
    setSaving(true);
    const profileData = {
      full_name: state.profile?.name || '',
      gender: state.profile?.gender || '',
      phone: state.profile?.phone || '',
      bio: state.profile?.bio || '',
      status: state.profile?.status || '',
    };
    await saveProfile(profileData);
    setSaving(false);
  };

  return (
    <section className='profile'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Профиль', 'Profile')}</div>
          <div className='panel-sub'>{state.profile?.email || ''}</div>
        </div>
        <div className='panel-body'>
          <div className='list-row'>
            <div>
              <div className='list-title'>{t('Язык', 'Language')}</div>
              <div className='muted'>
                {state.language === 'ru' ? 'RU' : 'EN'}
              </div>
            </div>
            <button className='secondary' onClick={toggleLanguage}>
              {t('Сменить', 'Switch')}
            </button>
          </div>
          <div className='list-row'>
            <div>
              <div className='list-title'>{t('Тема', 'Theme')}</div>
              <div className='muted'>{state.theme}</div>
            </div>
            <button className='secondary' onClick={toggleTheme}>
              {t('Сменить', 'Switch')}
            </button>
          </div>
          <div className='list-row'>
            <div>
              <div className='list-title'>
                {t('Высокая контрастность', 'High contrast')}
              </div>
              <div className='muted'>{state.highContrast ? 'ON' : 'OFF'}</div>
            </div>
            <button className='secondary' onClick={toggleHighContrast}>
              {t('Сменить', 'Switch')}
            </button>
          </div>
          <div className='list-row'>
            <div>
              <div className='list-title'>
                {t('Двухфакторная аутентификация', 'Two-Factor Authentication')}
              </div>
              <div className='muted'>
                {state.profile?.twoFactor ? 'ON' : 'OFF'}
              </div>
            </div>
            {!state.profile?.twoFactor ? (
              <button className='primary' onClick={setup2FA}>
                {t('Включить', 'Enable')}
              </button>
            ) : (
              <button
                className='danger'
                onClick={handleDisable}
                disabled={loading}
              >
                {loading
                  ? t('Отключение...', 'Disabling...')
                  : t('Отключить', 'Disable')}
              </button>
            )}
          </div>
          {state.profile?.twoFactor && (
            <div className='field'>
              <label>{t('Код для отключения', 'Code to disable')}</label>
              <input
                value={code}
                onChange={(e) => setCode(e.target.value)}
                maxLength={6}
              />
            </div>
          )}
          <button
            className='secondary full'
            onClick={handleSaveProfile}
            disabled={saving}
          >
            {saving
              ? t('Сохранение...', 'Saving...')
              : t('Сохранить профиль', 'Save profile')}
          </button>
          <button className='danger full' onClick={logout}>
            {t('Выйти', 'Logout')}
          </button>
          <button className='secondary full' onClick={() => go('home')}>
            {t('На главную', 'Go home')}
          </button>
        </div>
      </div>
    </section>
  );
}
