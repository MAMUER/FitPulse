import { useCallback, useEffect, useRef, useState } from 'react';
import Modal from './Modal';

const WORK_SEC = 45;
const REST_SEC = 20;

// WorkoutSession — экран активной тренировки: таймер «работа/отдых» по
// упражнениям, пауза, пропуск и завершение (начисляет XP через onFinish).
export default function WorkoutSession({ workout, t, onClose, onFinish }) {
  const exercises = workout?.exercises || [];
  const total = exercises.length;
  const [index, setIndex] = useState(0);
  const [phase, setPhase] = useState('work');
  const [remaining, setRemaining] = useState(WORK_SEC);
  const [paused, setPaused] = useState(false);
  const finishedRef = useRef(false);

  const finish = useCallback(() => {
    if (finishedRef.current) return;
    finishedRef.current = true;
    if (onFinish) onFinish();
    onClose();
  }, [onFinish, onClose]);

  // Автопереход между фазами и упражнениями.
  useEffect(() => {
    if (paused || total === 0) return undefined;
    if (remaining > 0) {
      const id = setTimeout(() => setRemaining((value) => value - 1), 1000);
      return () => clearTimeout(id);
    }
    // Фаза завершилась — переключаем.
    if (phase === 'work') {
      setPhase('rest');
      setRemaining(REST_SEC);
      return undefined;
    }
    if (index + 1 < total) {
      setIndex(index + 1);
      setPhase('work');
      setRemaining(WORK_SEC);
      return undefined;
    }
    // Тренировка завершена автоматически.
    finish();
    return undefined;
  }, [remaining, phase, paused, index, total, finish]);

  const current = exercises[index];
  const currentName = current ? current[0] : '';
  const currentDetail = current ? current[1] : '';
  const progress = total ? Math.round(((index + 1) / total) * 100) : 0;
  const isRest = phase === 'rest';

  return (
    <Modal
      className='fp-profile-modal'
      label={t('Тренировка', 'Workout')}
      onClose={onClose}
    >
      <section className='fp-profile-box'>
        <div className='between'>
          <div>
            <div className='eyebrow'>
              {t('ТРЕНИРОВКА', 'WORKOUT')} · {index + 1}/{total}
            </div>
            <h3>{workout?.title || t('Тренировка', 'Workout')}</h3>
          </div>
          <button
            type='button'
            className='icon'
            aria-label={t('Закрыть', 'Close')}
            onClick={onClose}
          >
            ×
          </button>
        </div>

        <div
          className='fp-main-number'
          style={{ justifyContent: 'center', textAlign: 'center' }}
        >
          <div>
            <strong style={{ fontSize: 44 }}>
              {String(Math.floor(remaining / 60)).padStart(1, '0')}:
              {String(remaining % 60).padStart(2, '0')}
            </strong>
            <small>{isRest ? t('отдых', 'rest') : t('работа', 'work')}</small>
          </div>
        </div>

        <div className='fp-progress-line'>
          <i style={{ width: `${progress}%` }} />
        </div>

        <div style={{ textAlign: 'center', margin: '10px 0' }}>
          <strong style={{ fontSize: 15 }}>{currentName}</strong>
          <div style={{ color: 'var(--muted)', fontSize: 11 }}>
            {currentDetail}
          </div>
        </div>

        <div className='form-row'>
          <button
            type='button'
            className='secondary'
            onClick={() => setPaused((value) => !value)}
          >
            {paused ? (
              <>
                <i className='fas fa-play' /> {t('Продолжить', 'Resume')}
              </>
            ) : (
              <>
                <i className='fas fa-pause' /> {t('Пауза', 'Pause')}
              </>
            )}
          </button>
          <button
            type='button'
            className='secondary'
            onClick={() => {
              if (phase === 'work') {
                setPhase('rest');
                setRemaining(REST_SEC);
              } else if (index + 1 < total) {
                setIndex(index + 1);
                setPhase('work');
                setRemaining(WORK_SEC);
              } else {
                finish();
              }
            }}
          >
            <i className='fas fa-forward' /> {t('Далее', 'Next')}
          </button>
        </div>

        <button
          type='button'
          className='primary full'
          style={{ marginTop: 12 }}
          onClick={finish}
        >
          <i className='fas fa-check' />{' '}
          {t('Завершить тренировку', 'Finish workout')}
        </button>
      </section>
    </Modal>
  );
}
