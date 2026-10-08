import { Navigate, Route, Routes } from 'react-router-dom';
import CookieConsent from './components/CookieConsent';
import { useApp } from './contexts/AppContext';
import Ai from './screens/AI';
import Body from './screens/Body';
import Calendar from './screens/Calendar';
import Chat from './screens/Chat';
import Confirm from './screens/Confirm';
import Home from './screens/Home';
import Integrations from './screens/Integrations';
import Landing from './screens/Landing';
import Legal from './screens/Legal';
import Login from './screens/Login';
import Menstrual from './screens/Menstrual';
import Nutrition from './screens/Nutrition';
import Profile from './screens/Profile';
import Register from './screens/Register';
import Reset from './screens/Reset';
import Training from './screens/Training';
import TwoFASetup from './screens/TwoFASetup';
import TwoFAVerify from './screens/TwoFAVerify';
import Videos from './screens/Videos';

function TabBar() {
  const { state, t, go } = useApp();
  const tabs = [
    { key: 'home', label: t('Главная', 'Home'), icon: 'fas fa-home' },
    {
      key: 'nutrition',
      label: t('Питание', 'Nutrition'),
      icon: 'fas fa-apple-alt',
    },
    {
      key: 'calendar',
      label: t('Календарь', 'Calendar'),
      icon: 'fas fa-calendar',
    },
    {
      key: 'training',
      label: t('Тренировки', 'Training'),
      icon: 'fas fa-dumbbell',
    },
    {
      key: 'integrations',
      label: t('Интеграции', 'Integrations'),
      icon: 'fas fa-microchip',
    },
    { key: 'ai', label: 'AI', icon: 'fas fa-robot' },
    { key: 'chat', label: t('Чат', 'Chat'), icon: 'fas fa-comment' },
    {
      key: 'menstrual',
      label: t('Цикл', 'Cycle'),
      icon: 'fas fa-venus',
    },
  ];

  return (
    <nav className='tab-bar'>
      {tabs.map((tab) => (
        <button
          type='button'
          key={tab.key}
          className={`tab${state.screen === tab.key ? ' active' : ''}`}
          onClick={() => go(tab.key)}
        >
          <i className={tab.icon}></i>
          <span>{tab.label}</span>
        </button>
      ))}
    </nav>
  );
}

export default function App() {
  const { state } = useApp();
  const authScreens = ['login', 'register', 'reset'];

  if (authScreens.includes(state.screen)) {
    return (
      <Routes>
        <Route path='/login' element={<Login />} />
        <Route path='/register' element={<Register />} />
        <Route path='/reset' element={<Reset />} />
        <Route path='/' element={<Navigate to='/login' replace />} />
        <Route path='*' element={<Navigate to='/login' replace />} />
      </Routes>
    );
  }

  return (
    <>
      <Routes>
        <Route path='/' element={<Landing />} />
        <Route path='/home' element={<Home />} />
        <Route path='/nutrition' element={<Nutrition />} />
        <Route path='/calendar' element={<Calendar />} />
        <Route path='/training' element={<Training />} />
        <Route path='/integrations' element={<Integrations />} />
        <Route path='/videos' element={<Videos />} />
        <Route path='/ai' element={<Ai />} />
        <Route path='/body' element={<Body />} />
        <Route path='/menstrual' element={<Menstrual />} />
        <Route path='/profile' element={<Profile />} />
        <Route path='/chat' element={<Chat />} />
        <Route path='/legal' element={<Legal />} />
        <Route path='/privacy' element={<Legal />} />
        <Route path='/terms' element={<Legal />} />
        <Route path='/consent' element={<Legal />} />
        <Route path='/twofa-setup' element={<TwoFASetup />} />
        <Route path='/twofa-verify' element={<TwoFAVerify />} />
        <Route path='/confirm' element={<Confirm />} />
        <Route path='*' element={<Navigate to='/' replace />} />
      </Routes>
      <CookieConsent />
      <TabBar />
    </>
  );
}
