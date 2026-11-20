import { useCallback, useEffect, useState } from 'react';
import EmptyState from '../components/EmptyState';
import ListItem from '../components/ListItem';
import Panel from '../components/Panel';
import { useApp } from '../contexts/AppContext';

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
      // Реального connect-API нет: синхронизация устройств управляется на
      // стороне OpenWearables (агрегатор Apple Health / Google Fit и т.д.).
      notify(
        t(
          `Чтобы подключить «${provider.source_name || source}», откройте OpenWearables и привяжите устройство/сервис там. FitPulse получает данные через OpenWearables автоматически.`,
          `To connect "${provider.source_name || source}", open OpenWearables and link the device/service there. FitPulse receives data through OpenWearables automatically.`
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
    if (providers.length === 0) {
      return (
        <EmptyState text={t('Нет доступных интеграций', 'No integrations')} />
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
                'FitPulse использует Open Wearables — агрегатор данных с фитнес-устройств и приложений. Чтобы подключить: откройте Open Wearables, привяжите источник (по OAuth), и FitPulse начнёт получать пульс, шаги, сон, SpO2 автоматически.',
                'FitPulse uses Open Wearables — an aggregator for fitness devices and apps. To connect: open Open Wearables, link the source (via OAuth), and FitPulse will start receiving heart rate, steps, sleep and SpO2 automatically.'
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
                  <span className='chip' key={name}>
                    {name}
                  </span>
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
