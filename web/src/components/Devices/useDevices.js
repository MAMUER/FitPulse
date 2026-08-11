import { useCallback, useEffect, useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { disconnectIntegration, getProviders } from '../../utils/api';

const OPEN_WEARABLES_WIDGET_URL =
  'https://cdn.openwearables.com/widget/v1/embed.js';
const OPEN_WEARABLES_APP_ID =
  process.env.REACT_APP_OPEN_WEARABLES_APP_ID || 'fitpulse-app';

export function useDevices() {
  const { token } = useAuth();
  const [status, setStatus] = useState('idle');
  const [error, setError] = useState('');
  const [providers, setProviders] = useState([]);

  useEffect(() => {
    if (!token) return;

    if (!document.getElementById('open-wearables-widget-script')) {
      const script = document.createElement('script');
      script.id = 'open-wearables-widget-script';
      script.src = OPEN_WEARABLES_WIDGET_URL;
      script.async = true;
      script.onerror = () => {
        setStatus('error');
        setError('Не удалось загрузить виджет Open Wearables');
      };
      document.body.appendChild(script);
    }

    const handleMessage = (event) => {
      const allowedOrigins = [
        'https://openwearables.com',
        'https://cdn.openwearables.com',
      ];

      if (!allowedOrigins.includes(event.origin)) {
        return;
      }

      const { type, data } = event.data || {};

      if (type === 'OPEN_WEARABLES_CONNECTED') {
        setStatus('connected');
        loadProviders();
      } else if (type === 'OPEN_WEARABLES_ERROR') {
        setStatus('error');
        setError(data?.message || 'Ошибка подключения');
      } else if (type === 'OPEN_WEARABLES_CLOSED') {
        setStatus('idle');
      }
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, [token]);

  const loadProviders = useCallback(async () => {
    try {
      const data = await getProviders();
      setProviders(data.providers || []);
    } catch (e) {
      console.error('Failed to load providers:', e);
    }
  }, [token]);

  const handleConnect = useCallback(() => {
    setStatus('loading');
    setError('');

    if (window.OpenWearablesWidget) {
      window.OpenWearablesWidget.init({
        appId: OPEN_WEARABLES_APP_ID,
        userId: getUserIdFromToken(token),
        onSuccess: () => {
          setStatus('connected');
          loadProviders();
        },
        onError: (err) => {
          setStatus('error');
          setError(err.message || 'Ошибка подключения');
        },
        onClose: () => {
          setStatus('idle');
        },
      });
    } else {
      /* istanbul ignore next */
      setTimeout(() => {
        if (window.OpenWearablesWidget) {
          handleConnect();
        } else {
          setStatus('error');
          setError('Виджет не загрузился. Попробуйте позже.');
        }
      }, 1000);
    }
  }, [token, loadProviders]);

  const handleDisconnect = async (source) => {
    if (!confirm(`Отключить источник ${source}?`)) return;
    try {
      await disconnectIntegration(source);
      loadProviders();
    } catch (e) {
      alert(`Ошибка отключения: ${e.message}`);
    }
  };

  return {
    status,
    error,
    providers,
    handleConnect,
    handleDisconnect,
  };
}

function getUserIdFromToken(jwtToken) {
  try {
    const payload = JSON.parse(atob(jwtToken.split('.')[1]));
    return payload.sub || payload.user_id;
  } catch {
    return 'anonymous';
  }
}
