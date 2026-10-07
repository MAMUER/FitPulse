import Panel from '../components/Panel';
import ListItem from '../components/ListItem';
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
      <Panel
        title={t('Профиль', 'Profile')}
        subtitle={state.profile?.email || ''}
      >
        <div className='panel-body'>
          <ListItem
            title={t('Язык', 'Language')}
            subtitle={state.language === 'ru' ? 'RU' : 'EN'}
            right={
              <button type='button' className='secondary' onClick={toggleLanguage}>
                {t('Сменить', 'Switch')}
              </button>
            }
          />
          <ListItem
            title={t('Тема', 'Theme')}
            subtitle={state.theme}
            right={
              <button type='button' className='secondary' onClick={toggleTheme}>
                {t('Сменить', 'Switch')}
              </button>
            }
          />
          <ListItem
            title={t('Высокая контрастность', 'High contrast')}
            subtitle={state.highContrast ? 'ON' : 'OFF'}
            right={
              <button type='button' className='secondary' onClick={toggleHighContrast}>
                {t('Сменить', 'Switch')}
              </button>
            }
          />
          <ListItem
            title={t('Двухфакторная аутентификация', 'Two-Factor Authentication')}
            subtitle={state.profile?.twoFactor ? 'ON' : 'OFF'}
            right={
              !state.profile?.twoFactor ? (
                <button type='button' className='primary' onClick={setup2FA}>
                  {t('Включить', 'Enable')}
                </button>
              ) : (
                <button
                  type='button'
                  className='danger'
                  onClick={handleDisable}
                  disabled={loading}
                >
                  {loading
                    ? t('Отключение...', 'Disabling...')
                    : t('Отключить', 'Disable')}
                </button>
              )
            }
          />
          {state.profile?.twoFactor && (
            <div className='field'>
              <label htmlFor='disableCode'>
                {t('Код для отключения', 'Code to disable')}
              </label>
              <input
                id='disableCode'
                value={code}
                onChange={(e) => setCode(e.target.value)}
                maxLength={6}
              />
            </div>
          )}
          <button
            type='button'
            className='secondary full'
            onClick={handleSaveProfile}
            disabled={saving}
          >
            {saving
              ? t('Сохранение...', 'Saving...')
              : t('Сохранить профиль', 'Save profile')}
          </button>
          <button type='button' className='danger full' onClick={logout}>
            {t('Выйти', 'Logout')}
          </button>
          <button
            type='button'
            className='secondary full'
            onClick={() => go('home')}
          >
            {t('На главную', 'Go home')}
          </button>
        </div>
      </Panel>
    </section>
  );
}
