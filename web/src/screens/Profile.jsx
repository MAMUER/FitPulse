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
  } = useApp();

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
