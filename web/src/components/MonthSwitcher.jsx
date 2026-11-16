import { useApp } from '../contexts/AppContext';

/**
 * Month label with prev/next controls.
 *
 * Home and Calendar rendered the same ~24-line block (eyebrow + two buttons);
 * only the weekday source differs, so that part stays with the caller.
 */
export default function MonthSwitcher({ label, onPrev, onNext }) {
  const { t } = useApp();

  return (
    <div className='between'>
      <span className='eyebrow'>{label}</span>
      <div className='row' style={{ gap: 6 }}>
        <button
          type='button'
          className='secondary'
          aria-label={t('Предыдущий месяц', 'Previous month')}
          onClick={onPrev}
        >
          <i className='fas fa-chevron-left' />
        </button>
        <button
          type='button'
          className='secondary'
          aria-label={t('Следующий месяц', 'Next month')}
          onClick={onNext}
        >
          <i className='fas fa-chevron-right' />
        </button>
      </div>
    </div>
  );
}
