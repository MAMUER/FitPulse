import { useDevices } from './useDevices';
import './Devices.css';

export default function Devices() {
  const { status, error, providers, handleConnect, handleDisconnect } =
    useDevices();

  return (
    <div className='view active'>
      <h3>Источники здоровья</h3>

      <div className='integration-status' aria-live='polite' aria-atomic='true'>
        {status === 'connected' && (
          <output className='success-message'>
            ✅ Успешно подключено! Данные будут синхронизироваться
            автоматически.
          </output>
        )}
        {status === 'loading' /* istanbul ignore next */ && (
          <output className='loading-message'>
            ⏳ Подключение к Open Wearables...
          </output>
        )}
        {status === 'error' && (
          <div className='error-message' role='alert'>
            ❌ {error}
          </div>
        )}
      </div>

      <button
        type='button'
        className='action-btn'
        onClick={handleConnect}
        disabled={status === 'loading'}
        aria-describedby='connect-help'
      >
        {status === 'loading' /* istanbul ignore next */
          ? 'Подключение...'
          : 'Подключить источники здоровья'}
      </button>
      <span id='connect-help' className='sr-only'>
        Подключает новые источники здоровья и носимые устройства.
      </span>

      <div className='connected-sources'>
        <h4>Подключённые источники</h4>
        {providers.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>
            Нет подключённых источников
          </p>
        ) : (
          <ul
            className='devices-list'
            aria-label='Список подключённых источников здоровья'
          >
            {providers.map((p) => (
              <li key={p.source} className='source-card'>
                <div className='source-info'>
                  <h4>{p.source_name || p.source}</h4>
                  <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
                    Подключён:{' '}
                    {new Date(p.connected_at).toLocaleDateString('ru-RU')}
                  </p>
                </div>
                <button
                  type='button'
                  onClick={() => handleDisconnect(p.source)}
                  className='btn-secondary'
                  style={{ padding: '8px 12px', fontSize: 13 }}
                  aria-label={`Отключить источник ${p.source_name || p.source}`}
                >
                  Отключить
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
