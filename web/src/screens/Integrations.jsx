import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';
import Panel from '../components/Panel';
import ListItem from '../components/ListItem';
import EmptyState from '../components/EmptyState';

export default function Integrations() {
  const { notify, t, loadIntegrationProviders, disconnectIntegration } = useApp();
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const data = await loadIntegrationProviders();
      setProviders(data);
    } catch {
      setProviders([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [loadIntegrationProviders]);

  const toggle = async (source) => {
    setActionLoading(source);
    try {
      await disconnectIntegration(source);
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

  const renderIntegrations = () => {
    if (loading) {
      return <EmptyState text={t('Загрузка...', 'Loading...')} />;
    }
    if (providers.length === 0) {
      return (
        <EmptyState
          text={t('Нет доступных интеграций', 'No integrations')}
        />
      );
    }
      return (
        <>
          {providers.map((provider) => {
            const isConnected = provider.connected;
            const isProcessing = actionLoading === provider.source;
            const statusText = isConnected
              ? t('Подключено', 'Connected')
              : t('Не подключено', 'Not connected');

            let buttonLabel;
            if (isProcessing) {
              buttonLabel = t('Обработка...', 'Processing...');
            } else if (isConnected) {
              buttonLabel = t('Отключить', 'Disconnect');
            } else {
              buttonLabel = t('Подключить', 'Connect');
            }

            const subtitle =
              statusText +
              (provider.connected_at
                ? ` · ${new Date(provider.connected_at).toLocaleString()}`
                : '');

            return (
              <ListItem
                key={provider.source}
                title={provider.source_name || provider.source}
                subtitle={subtitle}
                right={
                  <button
                    type='button'
                    className={isConnected ? 'danger' : 'primary'}
                    disabled={isProcessing}
                    onClick={() => toggle(provider.source)}
                  >
                    {buttonLabel}
                  </button>
                }
              />
            );
          })}
        </>
      );
  };

  return (
    <section className='integrations'>
      <Panel
        title={t('Интеграции', 'Integrations')}
        subtitle={t('Устройства и сервисы', 'Devices & services')}
      >
        <div className='panel-body'>{renderIntegrations()}</div>
      </Panel>
    </section>
  );
}
