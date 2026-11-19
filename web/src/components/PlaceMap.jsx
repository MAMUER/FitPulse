import { useApp } from '../contexts/AppContext';

const PLACES = [
  { name: 'Дом', icon: 'fa-house' },
  { name: 'Работа', icon: 'fa-building' },
  { name: 'Бассейн', icon: 'fa-person-swimming' },
  { name: 'Зал', icon: 'fa-dumbbell' },
  { name: 'Учеба', icon: 'fa-graduation-cap' },
];

// Выбор места тренировки. Карта (iframe) и «закреплённые точки» через prompt
// были нерабочими и не несли пользы для фитнес-приложения, поэтому оставлен
// простой селектор локации.
export default function PlaceMap() {
  const { state, selectPlace } = useApp();
  const selected = state.selectedPlace || 'Дом';

  return (
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
  );
}