import { useEffect, useState } from 'react';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import Modal from './components/Modal';
import Toast from './components/Toast';
import { useApp } from './contexts/AppContext';
import Ai from './screens/AI';
import Body from './screens/Body';
import Calendar from './screens/Calendar';
import Confirm from './screens/Confirm';
import Home from './screens/Home';
import Integrations from './screens/Integrations';
import Leaderboard from './screens/Leaderboard';
import Legal from './screens/Legal';
import Login from './screens/Login';
import Menstrual from './screens/Menstrual';
import Nutrition from './screens/Nutrition';
import Profile from './screens/Profile';
import Register from './screens/Register';
import Reset from './screens/Reset';
import Survey from './screens/Survey';
import Training from './screens/Training';
import TwoFASetup from './screens/TwoFASetup';
import TwoFAVerify from './screens/TwoFAVerify';

function TabBar() {
  const { t, go } = useApp();
  const { pathname } = useLocation();
  const activeKey = pathname.replace(/^\//, '').split('/')[0] || 'home';
  const [moreOpen, setMoreOpen] = useState(false);
  const navItem = (key, ru, en, icon) => ({ key, label: t(ru, en), icon });
  const tabs = [
    navItem('home', 'Главная', 'Home', 'fas fa-house'),
    navItem('nutrition', 'Питание', 'Nutrition', 'fas fa-utensils'),
    navItem('training', 'Тренировки', 'Training', 'fas fa-dumbbell'),
    navItem('ai', 'AI', 'AI', 'fas fa-robot'),
    navItem('calendar', 'Календарь', 'Calendar', 'fas fa-calendar-alt'),
  ];
  // Дополнительные экраны, которые не помещаются в нижнюю панель, доступны
  // через меню «Ещё». Раньше их не было в навигации вообще — до менструации и
  // интеграций (OpenWearables) нельзя было дойти.
  const moreItems = [
    navItem('leaderboard', 'Лидеры', 'Leaders', 'fas fa-trophy'),
    navItem('body', 'Тело', 'Body', 'fas fa-heart-pulse'),
    navItem('menstrual', 'Цикл', 'Cycle', 'fas fa-droplet'),
    navItem('integrations', 'Устройства', 'Devices', 'fas fa-watch'),
    navItem('profile', 'Профиль', 'Profile', 'fas fa-user'),
  ];
  const isMoreActive = moreItems.some((item) => activeKey === item.key);

  const goTo = (key) => {
    setMoreOpen(false);
    go(key);
  };

  return (
    <nav className='bottom-nav'>
      {tabs.map((tab) => (
        <button
          type='button'
          key={tab.key}
          className={`nav${activeKey === tab.key ? ' active' : ''}`}
          onClick={() => goTo(tab.key)}
        >
          <i className={tab.icon}></i>
          <span>{tab.label}</span>
        </button>
      ))}
      <button
        type='button'
        className={`nav${isMoreActive ? ' active' : ''}`}
        aria-expanded={moreOpen}
        onClick={() => setMoreOpen((open) => !open)}
      >
        <i className='fas fa-ellipsis-h'></i>
        <span>{t('Ещё', 'More')}</span>
      </button>
      {moreOpen && (
        <>
          <button
            type='button'
            aria-hidden='true'
            tabIndex={-1}
            className='fp-nav-backdrop'
            onClick={() => setMoreOpen(false)}
          />
          <div className='fp-nav-more-menu'>
            {moreItems.map((item) => (
              <button
                type='button'
                key={item.key}
                className={activeKey === item.key ? 'active' : ''}
                onClick={() => goTo(item.key)}
              >
                <i className={item.icon}></i>
                <span>{item.label}</span>
              </button>
            ))}
          </div>
        </>
      )}
    </nav>
  );
}

export default function App() {
  const { state, t, closeMetric, closeAchievement, update, notify } = useApp();
  const { pathname } = useLocation();

  // Держим state.screen в синхроне с реальным маршрутом. Иначе при переходе
  // кнопкой «назад» в браузере нижняя панель подсвечивала старую вкладку, а
  // go() мог не срабатывать из-за устаревшего state.screen.
  useEffect(() => {
    const screen = pathname.replace(/^\//, '').split('/')[0] || 'home';
    if (state.screen !== screen) {
      update?.({ screen });
    }
  }, [pathname, state.screen, update]);

  // Гостю один раз за сессию показываем, почему действия записи недоступны.
  useEffect(() => {
    if (
      state.authenticated === false &&
      !sessionStorage.getItem('fp-guest-notified')
    ) {
      notify(
        t(
          'Вы в гостевом режиме: войдите, чтобы сохранять тренировки, питание и данные.',
          'Guest mode: sign in to save workouts, nutrition and data.'
        )
      );
      sessionStorage.setItem('fp-guest-notified', '1');
    }
  }, [state.authenticated, notify, t]);

  const authScreens = new Set(['login', 'register', 'reset']);
  const chromeScreens = new Set([
    '/home',
    '/nutrition',
    '/calendar',
    '/training',
    '/integrations',
    '/leaderboard',
    '/ai',
    '/body',
    '/menstrual',
    '/profile',
  ]);
  const showAppChrome = chromeScreens.has(pathname);

  return (
    <>
      <Routes>
        <Route
          path='/'
          element={
            state.authenticated ? (
              <Navigate to='/home' replace />
            ) : (
              <Navigate to='/login' replace />
            )
          }
        />
        <Route path='/login' element={<Login />} />
        <Route path='/register' element={<Register />} />
        <Route path='/reset' element={<Reset />} />
        <Route path='/survey' element={<Survey />} />
        <Route path='/home' element={<Home />} />
        <Route path='/nutrition' element={<Nutrition />} />
        <Route path='/calendar' element={<Calendar />} />
        <Route path='/training' element={<Training />} />
        <Route path='/integrations' element={<Integrations />} />
        <Route path='/leaderboard' element={<Leaderboard />} />
        <Route path='/ai' element={<Ai />} />
        <Route path='/body' element={<Body />} />
        <Route path='/menstrual' element={<Menstrual />} />
        <Route path='/profile' element={<Profile />} />
        <Route path='/legal' element={<Legal />} />
        <Route path='/privacy' element={<Legal />} />
        <Route path='/terms' element={<Legal />} />
        <Route path='/consent' element={<Legal />} />
        <Route path='/twofa-setup' element={<TwoFASetup />} />
        <Route path='/twofa-verify' element={<TwoFAVerify />} />
        <Route path='/confirm' element={<Confirm />} />
        <Route
          path='*'
          element={
            <Navigate
              to={authScreens.has(state.screen) ? '/login' : '/'}
              replace
            />
          }
        />
      </Routes>
      {showAppChrome && <TabBar />}
      {state.selectedMetric && state.metricInfo?.[state.selectedMetric] && (
        <Modal
          className='fp-profile-modal'
          label={t('Показатели', 'Metrics')}
          onClose={closeMetric}
        >
          <section className='fp-profile-box'>
            <div className='between'>
              <h2>{state.metricInfo[state.selectedMetric][0]}</h2>
              <button
                type='button'
                className='icon'
                aria-label={t('Закрыть', 'Close')}
                onClick={closeMetric}
              >
                ×
              </button>
            </div>
            <strong>{state.metricInfo[state.selectedMetric][1]}</strong>
            <p className='muted'>{state.metricInfo[state.selectedMetric][2]}</p>
            <span>{state.metricInfo[state.selectedMetric][3]}</span>
          </section>
        </Modal>
      )}
      {state.selectedAchievement && (
        <Modal
          className='fp-profile-modal'
          label={t('Достижение', 'Achievement')}
          onClose={closeAchievement}
        >
          <section className='fp-profile-box'>
            <div className='between'>
              <h2>{state.selectedAchievement.name}</h2>
              <button
                type='button'
                className='icon'
                aria-label={t('Закрыть', 'Close')}
                onClick={closeAchievement}
              >
                ×
              </button>
            </div>
            <p className='muted'>{state.selectedAchievement.desc}</p>
          </section>
        </Modal>
      )}
      <Toast />
    </>
  );
}
