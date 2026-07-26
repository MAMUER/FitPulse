import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { login as apiLogin, register as apiRegister, logout as apiLogout, getProfile } from '../utils/api';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem('authToken'));
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [isAdmin, setIsAdmin] = useState(false);

  const refreshProfile = useCallback(async () => {
    try {
      const profile = await getProfile();
      setUser(profile);
      if (profile.role === 'admin' || profile.profile?.role === 'admin') {
        setIsAdmin(true);
      }
    } catch (e) {
      console.error('Failed to load profile:', e);
    }
  }, []);

  useEffect(() => {
    if (token) {
      refreshProfile().finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, [token, refreshProfile]);

  const login = async (email, password) => {
    const data = await apiLogin(email, password);
    setToken(data.access_token);
    if (data.role === 'admin') setIsAdmin(true);
    return data;
  };

  const register = async (email, password, fullName) => {
    const data = await apiRegister(email, password, fullName);
    return data;
  };

  const logout = async () => {
    await apiLogout();
    setToken(null);
    setUser(null);
    setIsAdmin(false);
  };

  return (
    <AuthContext.Provider value={{ token, user, loading, isAdmin, login, register, logout, refreshProfile, setToken: (t) => setToken(t) }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
