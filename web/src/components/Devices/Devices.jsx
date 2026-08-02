import { useCallback, useEffect, useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import './Devices.css';

const OPEN_WEARABLES_WIDGET_URL =
  'https://cdn.openwearables.com/widget/v1/embed.js';
const OPEN_WEARABLES_APP_ID =
  process.env.REACT_APP_OPEN_WEARABLES_APP_ID || 'fitpulse-app';

export default function Devices() {
  const [status, setStatus] = useState('idle');
  const [error, setError] = useState('');
  const [providers, setProviders] = useState([]);
  const { token } = useAuth();

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
      setTimeout(() => {
        if (window.OpenWearablesWidget) {
          handleConnect();
        } else {
          setStatus('error');
          setError('Виджет не загрузился. Попробуйте позже.');
        }
      }, 1000);
    }
  }, [token]);

  const loadProviders = async () => {
    try {
      const res = await fetch('/api/v1/integrations/providers', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      setProviders(data.providers || []);
    } catch (e) {
      console.error('Failed to load providers:', e);
    }
  };

  const handleDisconnect = async (source) => {
    if (!confirm(`Отключить источник ${source}?`)) return;
    try {
      await disconnectIntegration(source);
      loadProviders();
    } catch (e) {
      alert(`Ошибка отключения: ${e.message}`);
    }
  };

  const getUserIdFromToken = (jwtToken) => {
    try {
      const payload = JSON.parse(atob(jwtToken.split('.')[1]));
      return payload.sub || payload.user_id;
    } catch {
      return 'anonymous';
    }
  };

  return (
    <div className='view active'>
      <h3>Источники здоровья</h3>

      <div className='integration-status'>
        {status === 'connected' && (
          <div className='success-message'>
            ✅ Успешно подключено! Данные будут синхронизироваться
            автоматически.
          </div>
        )}
        {status === 'loading' && (
          <div className='loading-message'>
            ⏳ Подключение к Open Wearables...
          </div>
        )}
        {status === 'error' && <div className='error-message'>❌ {error}</div>}
      </div>

      <button
        className='action-btn'
        onClick={handleConnect}
        disabled={status === 'loading'}
      >
        {status === 'loading'
          ? 'Подключение...'
          : 'Подключить источники здоровья'}
      </button>

      <div className='connected-sources'>
        <h4>Подключённые источники</h4>
        {providers.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>
            Нет подключённых источников
          </p>
        ) : (
          providers.map((p) => (
            <div key={p.source} className='source-card'>
              <div className='source-info'>
                <h4>{p.source_name || p.source}</h4>
                <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
                  Подключён:{' '}
                  {new Date(p.connected_at).toLocaleDateString('ru-RU')}
                </p>
              </div>
              <button
                onClick={() => handleDisconnect(p.source)}
                className='btn-secondary'
                style={{ padding: '8px 12px', fontSize: 13 }}
              >
                Отключить
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
