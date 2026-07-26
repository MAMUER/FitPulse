import { useEffect, useState } from 'react';
import { fitbitAuth, getProviders, withingsAuth } from '../../utils/api';
import './Devices.css';

export default function Devices() {
  const [providers, setProviders] = useState([]);
  const [selectedDevice, setSelectedDevice] = useState(null);
  const [connecting, setConnecting] = useState(false);

  const devices = [
    {
      type: 'fitbit',
      name: 'Fitbit',
      icon: '⌚',
      capabilities: 'Пульс, SpO₂, Сон, Шаги, HRV',
    },
    {
      type: 'garmin',
      name: 'Garmin',
      icon: '⌚',
      capabilities: 'Пульс, SpO₂, Сон, Шаги, HRV, Темп',
    },
    {
      type: 'withings',
      name: 'Withings',
      icon: '⚖️',
      capabilities: 'Вес, Пульс, SpO₂, Сон',
    },
  ];

  useEffect(() => {
    loadProviders();
  }, [loadProviders]);

  const loadProviders = async () => {
    try {
      const data = await getProviders();
      setProviders(data.providers || []);
    } catch (e) {
      console.error('Failed to load providers:', e);
    }
  };

  const handleConnect = async () => {
    if (!selectedDevice) {
      alert('Выберите устройство из списка выше');
      return;
    }
    setConnecting(true);
    try {
      if (selectedDevice === 'fitbit') {
        await fitbitAuth();
      } else if (selectedDevice === 'withings') {
        await withingsAuth();
      }
      await loadProviders();
    } catch (e) {
      alert(`Ошибка подключения: ${e.message}`);
    } finally {
      setConnecting(false);
    }
  };

  const handleDisconnect = async (provider) => {
    if (!confirm(`Отключить ${provider}?`)) return;
    try {
      await fetch(`/api/v1/devices/${provider}/disconnect`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('authToken')}`,
        },
      });
      await loadProviders();
    } catch (e) {
      alert(`Ошибка отключения: ${e.message}`);
    }
  };

  return (
    <div className='view active'>
      <h3>Подключённые устройства</h3>
      <div id='connectedDevicesList' className='devices-list'>
        {providers.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>
            Нет подключённых устройств
          </p>
        ) : (
          providers.map((p) => (
            <div key={p.provider} className='device-card'>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                }}
              >
                <div>
                  <h4>{p.provider}</h4>
                  <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
                    ID: {p.provider_user_id}
                    {p.last_sync_at && (
                      <>
                        <br />
                        Последняя синхронизация:{' '}
                        {new Date(p.last_sync_at).toLocaleString('ru-RU')}
                      </>
                    )}
                  </p>
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <span
                    style={{
                      padding: '4px 8px',
                      background: p.is_active
                        ? 'var(--green)'
                        : 'var(--text-tertiary)',
                      color: 'white',
                      borderRadius: 12,
                      fontSize: 12,
                    }}
                  >
                    {p.is_active ? 'Активен' : 'Отключён'}
                  </span>
                  {p.is_active && (
                    <button
                      onClick={() => handleDisconnect(p.provider)}
                      className='btn-secondary'
                      style={{ padding: '8px 12px', fontSize: 13 }}
                    >
                      Отключить
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      <div className='device-selector'>
        {devices.map((device) => (
          <div
            key={device.type}
            className={`device-option ${selectedDevice === device.type ? 'selected' : ''}`}
            onClick={() => setSelectedDevice(device.type)}
          >
            <div className='device-icon'>{device.icon}</div>
            <div className='device-name'>{device.name}</div>
            <div className='device-capabilities'>{device.capabilities}</div>
          </div>
        ))}
      </div>

      <button
        className='action-btn'
        onClick={handleConnect}
        disabled={connecting}
      >
        {connecting ? 'Подключение...' : 'Подключить устройство'}
      </button>

      <div
        style={{
          marginTop: 16,
          paddingTop: 16,
          borderTop: '1px solid var(--border)',
        }}
      >
        <h4
          style={{
            marginBottom: 12,
            fontSize: 14,
            color: 'var(--text-secondary)',
          }}
        >
          Интеграции
        </h4>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <button
            onClick={fitbitAuth}
            className='btn-secondary'
            style={{ padding: '8px 12px', fontSize: 13 }}
          >
            ⌚ Fitbit
          </button>
          <button
            onClick={withingsAuth}
            className='btn-secondary'
            style={{ padding: '8px 12px', fontSize: 13 }}
          >
            ⚖️ Withings
          </button>
        </div>
      </div>
    </div>
  );
}
