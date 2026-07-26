import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from '../../contexts/AuthContext';
import './Layout.css';

export default function Layout() {
  const { logout, isAdmin } = useAuth();
  const location = useLocation();

  const tabs = [
    { path: '/', icon: 'dashboard', label: 'Обзор', view: 'dashboard' },
    { path: '/profile', icon: 'profile', label: 'Профиль', view: 'profile' },
    {
      path: '/training',
      icon: 'training',
      label: 'Тренировки',
      view: 'training',
    },
    { path: '/devices', icon: 'devices', label: 'Устройства', view: 'devices' },
    {
      path: '/achievements',
      icon: 'achievements',
      label: 'Достижения',
      view: 'achievements',
    },
    { path: '/diet', icon: 'diet', label: 'Диета', view: 'diet' },
    { path: '/health', icon: 'health', label: 'Здоровье', view: 'health' },
  ];

  const getPageTitle = () => {
    const tab = tabs.find((t) => t.path === location.pathname);
    if (tab) return tab.label;
    if (location.pathname === '/admin') return 'Админка';
    if (location.pathname === '/ml') return 'AI Анализ';
    return 'FitPulse';
  };

  return (
    <div className='app-layout'>
      <header className='top-bar'>
        <h2>{getPageTitle()}</h2>
        <button
          id='logoutBtn'
          className='btn-icon'
          onClick={logout}
          aria-label='Выйти'
        >
          <svg
            width='24'
            height='24'
            viewBox='0 0 24 24'
            fill='none'
            stroke='currentColor'
            strokeWidth='2'
          >
            <path d='M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4' />
            <polyline points='16 17 21 12 16 7' />
            <line x1='21' y1='12' x2='9' y2='12' />
          </svg>
        </button>
      </header>
      <main className='content'>
        <Outlet />
      </main>
      <nav className='tab-bar'>
        {tabs.map((tab) => (
          <NavLink
            key={tab.path}
            to={tab.path}
            className={({ isActive }) => `tab ${isActive ? 'active' : ''}`}
          >
            {tab.icon === 'dashboard' && (
              <svg
                width='24'
                height='24'
                viewBox='0 0 24 24'
                fill='none'
                stroke='currentColor'
                strokeWidth='2'
              >
                <rect x='3' y='3' width='7' height='7' rx='1' />
                <rect x='14' y='3' width='7' height='7' rx='1' />
                <rect x='3' y='14' width='7' height='7' rx='1' />
                <rect x='14' y='14' width='7' height='7' rx='1' />
              </svg>
            )}
            {tab.icon === 'profile' && (
              <svg
                width='24'
                height='24'
                viewBox='0 0 24 24'
                fill='none'
                stroke='currentColor'
                strokeWidth='2'
              >
                <path d='M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2' />
                <circle cx='12' cy='7' r='4' />
              </svg>
            )}
            {tab.icon === 'training' && (
              <svg
                width='24'
                height='24'
                viewBox='0 0 24 24'
                fill='none'
                stroke='currentColor'
                strokeWidth='2'
              >
                <polygon points='13 2 3 14 12 14 11 22 21 10 12 10 13 2' />
              </svg>
            )}
            {tab.icon === 'devices' && (
              <svg
                width='24'
                height='24'
                viewBox='0 0 24 24'
                fill='none'
                stroke='currentColor'
                strokeWidth='2'
              >
                <rect x='5' y='2' width='14' height='20' rx='2' />
                <line x1='12' y1='18' x2='12' y2='18.01' />
              </svg>
            )}
            {tab.icon === 'achievements' && (
              <svg
                width='24'
                height='24'
                viewBox='0 0 24 24'
                fill='none'
                stroke='currentColor'
                strokeWidth='2'
              >
                <circle cx='12' cy='8' r='7' />
                <polyline points='8.21 13.89 7 23 12 20 17 23 15.79 13.88' />
              </svg>
            )}
            {tab.icon === 'diet' && (
              <svg
                width='24'
                height='24'
                viewBox='0 0 24 24'
                fill='none'
                stroke='currentColor'
                strokeWidth='2'
              >
                <path d='M18 8h1a4 4 0 010 8h-1' />
                <path d='M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z' />
                <line x1='6' y1='1' x2='6' y2='4' />
                <line x1='10' y1='1' x2='10' y2='4' />
                <line x1='14' y1='1' x2='14' y2='4' />
              </svg>
            )}
            {tab.icon === 'health' && (
              <svg
                width='24'
                height='24'
                viewBox='0 0 24 24'
                fill='none'
                stroke='currentColor'
                strokeWidth='2'
              >
                <path d='M20.84 4.61a5.5 5.5 0 00-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 00-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 000-7.78z' />
              </svg>
            )}
            <span>{tab.label}</span>
          </NavLink>
        ))}
        {isAdmin && (
          <NavLink
            to='/admin'
            className={({ isActive }) => `tab ${isActive ? 'active' : ''}`}
          >
            <svg
              width='24'
              height='24'
              viewBox='0 0 24 24'
              fill='none'
              stroke='currentColor'
              strokeWidth='2'
            >
              <path d='M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z' />
            </svg>
            <span>Админка</span>
          </NavLink>
        )}
      </nav>
    </div>
  );
}
