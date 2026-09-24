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

  const getButtonLabel = (isProcessing, isConnected) => {
    if (isProcessing) {
      return t('Обработка...', 'Processing...');
    }
    if (isConnected) {
      return t('Отключить', 'Disconnect');
    }
    return t('Подключить', 'Connect');
  };

  const getStatusText = (isConnected) => {
    return isConnected
      ? t('Подключено', 'Connected')
      : t('Не подключено', 'Not connected');
  };

  const getTimestamp = (connectedAt) => {
    return connectedAt ? ` · ${new Date(connectedAt).toLocaleString()}` : '';
  };

  const renderProvider = (provider) => {
    const isConnected = provider.connected;
    const isProcessing = actionLoading === provider.source;
    return (
      <div className='list-row' key={provider.source}>
        <div>
          <div className='list-title'>
            {provider.source_name || provider.source}
          </div>
          <div className='muted'>
            {getStatusText(isConnected)}
            {getTimestamp(provider.connected_at)}
          </div>
        </div>
        <button
          type='button'
          className={isConnected ? 'danger' : 'primary'}
          disabled={isProcessing}
          onClick={() => toggle(provider.source)}
        >
          {getButtonLabel(isProcessing, isConnected)}
        </button>
      </div>
    );
  };

  const renderContent = () => {
    if (loading) {
      return <div className='muted'>{t('Загрузка...', 'Loading...')}</div>;
    }
    if (providers.length === 0) {
      return (
        <div className='muted'>
          {t('Нет доступных интеграций', 'No integrations')}
        </div>
      );
    }
    return <>{providers.map(renderProvider)}</>;
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
        <div className='panel-body'>{renderContent()}</div>
      </div>
    </section>
  );
}
