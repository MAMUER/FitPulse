import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';
import { backendRequest } from '../utils/backendRequest';

export default function Integrations() {
  const { notify, t } = useApp();
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const data = await backendRequest('/api/v1/integrations/providers');
      if (data && Array.isArray(data.providers)) {
        setProviders(data.providers);
      } else {
        setProviders([]);
      }
    } catch {
      setProviders([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const toggle = async (source) => {
    setActionLoading(source);
    try {
      await backendRequest(`/api/v1/integrations/${source}/disconnect`, {
        method: 'POST',
      });
      setProviders((prev) =>
        prev.map((p) =>
          p.source === source
            ? { ...p, connected: false, connected_at: null }
            : p
        )
      );
      notify('Интеграция отключена');
    } catch {
      notify('Не удалось изменить интеграцию');
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <section className='integrations'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Интеграции', 'Integrations')}</div>
          <div className='panel-sub'>
            {t('Устройства и сервисы', 'Devices & services')}
          </div>
        </div>
        <div className='panel-body'>
          {loading ? (
            <div className='muted'>{t('Загрузка...', 'Loading...')}</div>
          ) : providers.length === 0 ? (
            <div className='muted'>
              {t('Нет доступных интеграций', 'No integrations')}
            </div>
          ) : (
            providers.map((provider) => (
              <div className='list-row' key={provider.source}>
                <div>
                  <div className='list-title'>{provider.source_name || provider.source}</div>
                  <div className='muted'>
                    {provider.connected
                      ? t('Подключено', 'Connected')
                      : t('Не подключено', 'Not connected')}
                    {provider.connected_at
                      ? ` · ${new Date(provider.connected_at).toLocaleString()}`
                      : ''}
                  </div>
                </div>
                <button
                  className={provider.connected ? 'danger' : 'primary'}
                  disabled={actionLoading === provider.source}
                  onClick={() => toggle(provider.source)}
                >
                  {actionLoading === provider.source
                    ? t('Обработка...', 'Processing...')
                    : provider.connected
                      ? t('Отключить', 'Disconnect')
                      : t('Подключить', 'Connect')}
                </button>
              </div>
            ))
          )}
        </div>
      </div>
    </section>
  );
}
