import { useCallback, useEffect, useState } from 'react';
import EmptyState from '../components/EmptyState';
import ListItem from '../components/ListItem';
import Panel from '../components/Panel';
import { useApp } from '../contexts/AppContext';

// Если бэкенд не вернул провайдеров, показываем статичный список (реальное
// подключение — через OpenWearables).
const FALLBACK_PROVIDERS = [
  'Garmin',
  'Oura',
  'Whoop',
  'Polar',
  'Suunto',
  'Strava',
  'Withings',
  'Apple Health',
  'Google Health Connect',
  'Samsung Health',
].map((name) => ({
  source: name.toLowerCase().replace(/\s+/g, '-'),
  source_name: name,
  connected: false,
}));

export default function Integrations() {
  const { notify, t, loadIntegrationProviders, disconnectIntegration } =
    useApp();
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await loadIntegrationProviders();
      setProviders(data);
    } catch {
      setProviders([]);
    } finally {
      setLoading(false);
    }
  }, [loadIntegrationProviders]);

  useEffect(() => {
    void load();
  }, [load]);

  const toggle = async (provider) => {
    const source = provider.source;
    const isConnected = provider.connected;
    if (!isConnected) {
      // Реального connect-API нет: привязка выполняется в OpenWearables.
      // Открываем его через gateway (reverse proxy /open-wearables).
      const configured =
        import.meta.env.VITE_OPEN_WEARABLES_URL || '/open-wearables';
      const base = configured.endsWith('/')
        ? configured.slice(0, -1)
        : configured;
      window.open(
        `${base}/?source=${encodeURIComponent(source)}`,
        '_blank',
        'noopener,noreferrer'
      );
      notify(
        t(
          `Открываем OpenWearables для подключения «${provider.source_name || source}».`,
          `Opening OpenWearables to connect "${provider.source_name || source}".`
        )
      );
      return;
    }
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
      notify(t('Интеграция отключена', 'Integration disconnected'));
    } catch {
      notify(
        t('Не удалось изменить интеграцию', 'Failed to update integration')
      );
    } finally {
      setActionLoading(null);
    }
  };

  const renderIntegrations = () => {
    if (loading) {
      return <EmptyState text={t('Загрузка...', 'Loading...')} />;
    }
    const list = providers.length ? providers : FALLBACK_PROVIDERS;
    return (
      <>
        {list.map((provider) => {
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
                  onClick={() => toggle(provider)}
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
        <div className='panel-body'>
          <div className='card' style={{ marginBottom: 12 }}>
            <div className='eyebrow'>OPEN WEARABLES</div>
            <p
              className='muted'
              style={{ marginTop: 6, fontSize: 11, lineHeight: 1.5 }}
            >
              {t(
                'FitPulse использует Open Wearables — агрегатор данных с фитнес-устройств и приложений. Нажмите «Подключить» напротив источника: откроется Open Wearables, где нужно привязать устройство или сервис по OAuth. FitPulse начнёт получать пульс, шаги, сон и SpO2 автоматически.',
                'FitPulse uses Open Wearables — an aggregator for fitness devices and apps. Click "Connect" next to a source: Open Wearables opens, where you link the device or service via OAuth. FitPulse will start receiving heart rate, steps, sleep and SpO2 automatically.'
              )}
            </p>
            <div className='fp-provider-chips' style={{ marginTop: 8 }}>
              <div className='eyebrow' style={{ marginBottom: 4 }}>
                {t('Облачные (OAuth)', 'Cloud (OAuth)')}
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                {[
                  'Garmin',
                  'Oura',
                  'Whoop',
                  'Polar',
                  'Suunto',
                  'Strava',
                  'Withings',
                  'Ultrahuman',
                  'SensorBio',
                  'Google Health',
                  'Fitbit (deprecated)',
                ].map((name) => (
                  <button
                    type='button'
                    className='chip'
                    key={name}
                    onClick={() =>
                      toggle({
                        source: name.toLowerCase().replace(/\s+/g, '-'),
                        source_name: name,
                        connected: false,
                      })
                    }
                  >
                    {name}
                  </button>
                ))}
              </div>
              <div
                className='eyebrow'
                style={{ marginBottom: 4, marginTop: 10 }}
              >
                {t('На устройстве (мобильный SDK)', 'On-device (mobile SDK)')}
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                {[
                  'Apple Health',
                  'Google Health Connect',
                  'Samsung Health',
                ].map((name) => (
                  <span className='chip' key={name}>
                    {name}
                  </span>
                ))}
              </div>
            </div>
          </div>
          {renderIntegrations()}
        </div>
      </Panel>
    </section>
  );
}
