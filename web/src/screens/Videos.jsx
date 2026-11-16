import { useEffect, useState } from 'react';
import VideoCard from '../components/VideoCard';
import VideoModal from '../components/VideoModal';
import { useApp } from '../contexts/AppContext';

const EQUIPMENT = [
  {
    icon: 'fa-circle',
    title: ['Коврик', 'Mat'],
    note: ['Для пола и растяжки', 'For floor and stretching'],
  },
  {
    icon: 'fa-dumbbell',
    title: ['Гантели', 'Dumbbells'],
    note: ['2 × 4–8 кг', '2 × 4–8 kg'],
  },
  {
    icon: 'fa-circle-notch',
    title: ['Резинка', 'Resistance band'],
    note: ['Активация мышц', 'Muscle activation'],
  },
  {
    icon: 'fa-stopwatch',
    title: ['Таймер', 'Timer'],
    note: ['Для интервалов', 'For intervals'],
  },
];

export default function Videos() {
  const { state, t, loadVideos } = useApp();
  const [videoIndex, setVideoIndex] = useState(null);

  useEffect(() => {
    loadVideos();
  }, [loadVideos]);

  const videos = state.trainingVideos || [];
  const library = state.videos || [];

  const changeVideo = (direction) => {
    if (!videos.length) return;
    setVideoIndex(
      (current) => (current + direction + videos.length) % videos.length
    );
  };

  return (
    <section className='videos'>
      <div className='page-title'>
        <div className='eyebrow'>FITPULSE MOTION</div>
        <h2>{t('Видео и оборудование', 'Videos and equipment')}</h2>
        <p>{t('Короткие тренировки для дома.', 'Short workouts for home.')}</p>
      </div>

      {videos.map((video, index) => (
        <VideoCard
          key={video.id || index}
          video={video}
          index={index}
          onSelect={setVideoIndex}
        />
      ))}

      {library.length > 0 && (
        <>
          <div className='section'>
            <h3>{t('Библиотека FitPulse', 'FitPulse library')}</h3>
          </div>
          <section className='card'>
            {library.map((item, index) => (
              <div className='body-row' key={item.id || index}>
                <span>{item.title}</span>
                <b>
                  {item.duration || item.minutes || ''} {t('мин', 'min')}
                </b>
              </div>
            ))}
          </section>
        </>
      )}

      <div className='section'>
        <h3>{t('Оборудование для дома', 'Home equipment')}</h3>
      </div>
      <section className='grid-2'>
        {EQUIPMENT.map((item) => (
          <article className='card equipment' key={item.icon}>
            <i className={`fas ${item.icon}`} />
            <strong>{t(...item.title)}</strong>
            <small>{t(...item.note)}</small>
          </article>
        ))}
      </section>

      {videoIndex !== null && (
        <VideoModal
          videos={videos}
          index={videoIndex}
          onChange={changeVideo}
          onClose={() => setVideoIndex(null)}
        />
      )}
    </section>
  );
}
