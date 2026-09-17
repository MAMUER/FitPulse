import { Navigate, Route, Routes } from 'react-router-dom';
import { useApp } from './contexts/AppContext';
import AI from './screens/AI';
import Body from './screens/Body';
import Calendar from './screens/Calendar';
import Chat from './screens/Chat';
import Home from './screens/Home';
import Legal from './screens/Legal';
import Login from './screens/Login';
import Nutrition from './screens/Nutrition';
import Profile from './screens/Profile';
import Register from './screens/Register';
import Reset from './screens/Reset';
import Training from './screens/Training';
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
    { key: 'ai', label: 'AI', icon: 'fas fa-robot' },
    { key: 'chat', label: t('Чат', 'Chat'), icon: 'fas fa-comment' },
  ];

  return (
    <nav className='tab-bar'>
      {tabs.map((tab) => (
        <button
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
        <Route path='*' element={<Login />} />
      </Routes>
    );
  }

  return (
    <>
      <Routes>
        <Route path='/' element={<Home />} />
        <Route path='/home' element={<Home />} />
        <Route path='/nutrition' element={<Nutrition />} />
        <Route path='/calendar' element={<Calendar />} />
        <Route path='/training' element={<Training />} />
        <Route path='/videos' element={<Videos />} />
        <Route path='/ai' element={<AI />} />
        <Route path='/body' element={<Body />} />
        <Route path='/profile' element={<Profile />} />
        <Route path='/chat' element={<Chat />} />
        <Route path='/legal' element={<Legal />} />
        <Route path='*' element={<Navigate to='/' replace />} />
      </Routes>
      <TabBar />
    </>
  );
}
