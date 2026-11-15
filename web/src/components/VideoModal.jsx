import Modal from './Modal';
import { useApp } from '../contexts/AppContext';

export default function VideoModal({ videos, index, onChange, onClose }) {
  const { t, notify } = useApp();
  const video = videos[index];

  if (!video) return null;

  const equipment = video.equipment || [];

  return (
    <Modal
      className='fp-video-modal'
      label={video.title}
      onClose={onClose}
    >
      <section className='fp-video-modal-box'>
        <div className='between'>
          <div className='eyebrow'>{t('ВИДЕО-УРОК', 'VIDEO LESSON')}</div>
          <button
            type='button'
            className='icon'
            aria-label={t('Закрыть', 'Close')}
            onClick={onClose}
          >
            ×
          </button>
        </div>
        <img
          className='fp-video-modal-image'
          src={video.image}
          alt={video.title}
        />
        <h2>{video.title}</h2>
        <p className='muted' style={{ marginTop: 7 }}>
          {video.description}
        </p>
        <div className='fp-video-info'>
          <div>
            <strong>{video.duration}</strong>
            <small>{t('длительность', 'duration')}</small>
          </div>
          <div>
            <strong>{video.level}</strong>
            <small>{t('уровень', 'level')}</small>
          </div>
          <div>
            <strong>{equipment.join(', ')}</strong>
            <small>{t('понадобится', 'equipment')}</small>
          </div>
        </div>
        <button
          type='button'
          className='primary full'
          style={{ marginTop: 14 }}
          onClick={() =>
            notify(
              t(
                'Видео-плеер подключается через внешний сервис.',
                'The video player is provided by an external service.'
              )
            )
          }
        >
          <i className='fas fa-play' />{' '}
          {t('Подключиться к видео', 'Connect to video')}
        </button>
        <div className='fp-video-actions'>
          <button
            type='button'
            className='secondary'
            aria-label={t('Предыдущее видео', 'Previous video')}
            onClick={() => onChange(-1)}
          >
            <i className='fas fa-chevron-left' />
          </button>
          <button
            type='button'
            className='secondary'
            aria-label={t('Следующее видео', 'Next video')}
            onClick={() => onChange(1)}
          >
            <i className='fas fa-chevron-right' />
          </button>
        </div>
      </section>
    </Modal>
  );
}
