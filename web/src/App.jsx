import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './contexts/AuthContext';
import Layout from './components/Layout/Layout';
import AuthScreen from './components/Auth/AuthScreen';
import Confirm from './components/Auth/Confirm';
import Dashboard from './components/Dashboard/Dashboard';
import Profile from './components/Profile/Profile';
import Training from './components/Training/Training';
import Devices from './components/Devices/Devices';
import Achievements from './components/Achievements/Achievements';
import Diet from './components/Diet/Diet';
import Health from './components/Health/Health';
import ML from './components/ML/ML';
import Admin from './components/Admin/Admin';

export default function App() {
  const { token, loading } = useAuth();

  if (loading) {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100dvh',
        background: 'var(--bg-primary)',
        color: 'var(--text-primary)',
        fontSize: '18px',
      }}>
        Загрузка...
      </div>
    );
  }

  if (!token) {
    return (
      <Routes>
        <Route path="/confirm" element={<Confirm />} />
        <Route path="*" element={<AuthScreen />} />
      </Routes>
    );
  }

  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/profile" element={<Profile />} />
        <Route path="/training" element={<Training />} />
        <Route path="/devices" element={<Devices />} />
        <Route path="/achievements" element={<Achievements />} />
        <Route path="/diet" element={<Diet />} />
        <Route path="/health" element={<Health />} />
        <Route path="/ml" element={<ML />} />
        <Route path="/admin" element={<Admin />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  );
}
