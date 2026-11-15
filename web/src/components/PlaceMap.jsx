import { useApp } from '../contexts/AppContext';

const PLACES = [
  { name: 'Дом', icon: 'fa-house' },
  { name: 'Работа', icon: 'fa-building' },
  { name: 'Бассейн', icon: 'fa-person-swimming' },
  { name: 'Зал', icon: 'fa-dumbbell' },
  { name: 'Учеба', icon: 'fa-graduation-cap' },
];

export default function PlaceMap() {
  const { state, t, selectPlace, addPlace, editPlace, notify } = useApp();
  const places = state.pinnedPlaces || [];
  const selected = state.selectedPlace || 'Дом';

  const openMap = () => {
    notify(
      t(
        'Интерактивная карта и геокодирование требуют картографического API. Открыт внешний просмотр.',
        'Interactive maps and geocoding require a maps API. Opening external view.'
      )
    );
    window.open('https://yandex.ru/maps', '_blank', 'noopener');
  };

  return (
    <>
      <section className='place-map'>
        <iframe
          title={t('Карта тренировок', 'Training map')}
          src='https://yandex.ru/map-widget/v1/?ll=37.617634,55.755826&z=12&l=map&pt=37.617634,55.755826,flag'
          allowFullScreen
          loading='lazy'
        />
        <div className='fp-map-pins'>
          {places.map((place, index) => (
            <button
              type='button'
              key={`${place.name}-${index}`}
              className='fp-map-pin'
              style={{
                left: `${place.x}%`,
                top: `${place.y}%`,
                '--pin-color': place.color,
              }}
              title={place.name}
              onClick={() => editPlace(index)}
            >
              <i className={`fas ${place.icon}`} />
              <span>{place.name}</span>
            </button>
          ))}
        </div>
      </section>
      <div className='place-buttons'>
        {PLACES.map((place) => (
          <button
            type='button'
            key={place.name}
            className={`place-button ${selected === place.name ? 'active' : ''}`}
            onClick={() => selectPlace(place.name)}
          >
            <i className={`fas ${place.icon}`} /> {place.name}
          </button>
        ))}
      </div>
      <div className='fp-place-editor'>
        <strong>{t('Закреплённые точки на карте', 'Pinned map places')}</strong>
        <small style={{ color: 'var(--muted)' }}>
          {t(
            'Нажмите на точку, чтобы изменить её положение.',
            'Click a pin to change its position.'
          )}
        </small>
        <button type='button' className='secondary full' onClick={addPlace}>
          <i className='fas fa-plus' /> {t('Добавить точку', 'Add place')}
        </button>
      </div>
      <p style={{ marginTop: 8, color: 'var(--muted)', fontSize: 9 }}>
        {t('Выбрано', 'Selected')}:{' '}
        <b style={{ color: 'var(--blue)' }}>{selected}</b>
        {' · '}
        <button type='button' className='link' onClick={openMap}>
          {t('Открыть карту', 'Open map')}
        </button>
      </p>
    </>
  );
}
