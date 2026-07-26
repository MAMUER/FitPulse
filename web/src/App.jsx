import { Navigate, Route, Routes } from 'react-router-dom';
import Achievements from './components/Achievements/Achievements';
import Admin from './components/Admin/Admin';
import AuthScreen from './components/Auth/AuthScreen';
import Confirm from './components/Auth/Confirm';
import Dashboard from './components/Dashboard/Dashboard';
import Devices from './components/Devices/Devices';
import Diet from './components/Diet/Diet';
import Health from './components/Health/Health';
import Layout from './components/Layout/Layout';
import ML from './components/ML/ML';
import Profile from './components/Profile/Profile';
import Training from './components/Training/Training';
import { useAuth } from './contexts/AuthContext';

export default function App() {
  const { token, loading } = useAuth();

  if (loading) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          minHeight: '100dvh',
          background: 'var(--bg-primary)',
          color: 'var(--text-primary)',
          fontSize: '18px',
        }}
      >
        Загрузка...
      </div>
    );
  }

  if (!token) {
    return (
      <Routes>
        <Route path='/confirm' element={<Confirm />} />
        <Route path='*' element={<AuthScreen />} />
      </Routes>
    );
  }

  return (
    <Layout>
      <Routes>
        <Route path='/' element={<Dashboard />} />
        <Route path='/profile' element={<Profile />} />
        <Route path='/training' element={<Training />} />
        <Route path='/devices' element={<Devices />} />
        <Route path='/achievements' element={<Achievements />} />
        <Route path='/diet' element={<Diet />} />
        <Route path='/health' element={<Health />} />
        <Route path='/ml' element={<ML />} />
        <Route path='/admin' element={<Admin />} />
        <Route path='*' element={<Navigate to='/' replace />} />
      </Routes>
    </Layout>
  );
}
