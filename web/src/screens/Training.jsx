import { useEffect, useState } from 'react';
import VideoCard from '../components/VideoCard';
import VideoModal from '../components/VideoModal';
import WorkoutSession from '../components/WorkoutSession';
import { useApp } from '../contexts/AppContext';

export default function Training() {
  const {
    state,
    t,
    canEdit,
    formatRelativeDate,
    startWorkout,
    toggleWorkout,
    setTrainingPeriod,
    loadTrainingPlans,
    generatePlan,
    customExercises,
    addCustomExercise,
    removeCustomExercise,
    quickAI,
    go,
  } = useApp();

  const [videoIndex, setVideoIndex] = useState(null);
  const [exerciseName, setExerciseName] = useState('');
  const [exerciseSets, setExerciseSets] = useState('');
  const [activeWorkout, setActiveWorkout] = useState(null);

  useEffect(() => {
    loadTrainingPlans();
  }, [loadTrainingPlans]);

  const period = state.trainingPeriod || 'day';
  const data = state.trainingData?.[period] || {
    completed: 0,
    goal: 0,
    calories: 0,
    minutes: 0,
    workouts: [],
  };
  // Значения берём из trainingData (счётчик приложения, обновляется при
  // завершении тренировок через startWorkout).
  const completedCount = data.completed || 0;
  const goalCount = data.goal || 0;
  const realCalories = data.calories || 0;
  const realMinutes = data.minutes || 0;
  const workoutNames = data.workouts || [];
  const percentage = goalCount
    ? Math.min(100, Math.round((completedCount / goalCount) * 100))
    : 0;
  const completedWorkouts = state.completedWorkouts || [];
  const exercises = customExercises || [];
  const videos = state.trainingVideos || [];
  const plans = state.trainingPlans || [];
  const instructions = state.instructions || {};

  const addExercise = () => {
    const name = exerciseName.trim();
    const sets = exerciseSets.trim();
    if (!name || !sets || !addCustomExercise) return;
    addCustomExercise({ name, sets });
    setExerciseName('');
    setExerciseSets('');
  };

  const handleVideoChange = (direction) => {
    if (!videos.length) return;
    setVideoIndex(
      (current) => (current + direction + videos.length) % videos.length
    );
  };

  return (
    <section className='training'>
      <div className='page-title'>
        <div className='eyebrow'>ADAPTIVE TRAINING</div>
        <h2>{t('Тренировки', 'Training')}</h2>
        <p>
          {t(
            'Дата, время, интервалы, упражнения и расход калорий.',
            'Date, time, intervals, exercises and calorie burn.'
          )}
        </p>
      </div>

      <section className='fp-training-panel'>
        <h3>{t('План тренировок', 'Training plan')}</h3>
        <div className='fp-period-tabs'>
          {[
            ['day', t('День', 'Day')],
            ['week', t('Неделя', 'Week')],
            ['month', t('Месяц', 'Month')],
          ].map(([key, label]) => (
            <button
              type='button'
              key={key}
              className={period === key ? 'active' : ''}
              onClick={() => setTrainingPeriod(key)}
            >
              {label}
            </button>
          ))}
        </div>
        <div className='fp-main-number'>
          <div>
            <strong>
              {completedCount}/{goalCount}
            </strong>
            <small>{t('выполнено тренировок', 'workouts completed')}</small>
          </div>
          <small>
            {realCalories} {t('ккал', 'kcal')}
          </small>
        </div>
        <div className='fp-progress-line'>
          <i style={{ width: `${percentage}%` }} />
        </div>
        <div className='fp-stats'>
          <div className='fp-stat'>
            <strong>{realMinutes}</strong>
            <small>{t('минут', 'minutes')}</small>
          </div>
          <div className='fp-stat'>
            <strong>{realCalories}</strong>
            <small>{t('ккал', 'kcal')}</small>
          </div>
          <div className='fp-stat'>
            <strong>{goalCount}</strong>
            <small>{t('цель', 'goal')}</small>
          </div>
        </div>
        <div className='fp-training-list'>
          {workoutNames.map((workout, index) => (
            <div className='fp-training-item' key={`${workout}-${index}`}>
              <i
                className={`fas ${
                  index < completedCount ? 'fa-circle-check' : 'fa-circle'
                }`}
              />
              <div>
                <strong>{workout}</strong>
                <small>
                  {index < completedCount
                    ? t('Выполнено', 'Completed')
                    : t('Запланировано', 'Scheduled')}
                </small>
              </div>
              {index < completedCount ? <span>+40 XP</span> : null}
            </div>
          ))}
        </div>
      </section>

      <div className='section'>
        <h3>{t('Мой план', 'My plan')}</h3>
      </div>
      {plans.map((workout) => {
        const title = t(...workout.title);
        const completed = completedWorkouts.includes(title);
        const open = state.selectedWorkout === workout.id;
        return (
          <article
            className={`workout-card ${open ? 'open' : ''} ${
              completed ? 'completed' : ''
            }`}
            key={workout.id}
          >
            <div className='workout-title'>
              <strong>
                {completed ? '✓ ' : ''}
                {title}
              </strong>
              <small>{workout.time}</small>
            </div>
            <p style={{ marginTop: 5, color: 'var(--muted)', fontSize: 9 }}>
              {formatRelativeDate(workout.offset, state.language)}
            </p>
            <div className='workout-meta'>
              <span>
                <i className='fas fa-clock' /> {workout.duration}
              </span>
              <span>
                <i className='fas fa-fire' /> {workout.calories}{' '}
                {t('ккал', 'kcal')}
              </span>
              <span>
                <i className='fas fa-signal' /> {t(...workout.level)}
              </span>
            </div>
            <div className='interval'>
              <div>
                <b>45</b>
                <small>{t('сек работа', 'sec work')}</small>
              </div>
              <div>
                <b>20</b>
                <small>{t('сек отдых', 'sec rest')}</small>
              </div>
              <div>
                <b>{workout.exercises.length}</b>
                <small>{t('упражнений', 'exercises')}</small>
              </div>
            </div>
            <div className='exercise-list'>
              {workout.exercises.map((exercise) => (
                <div className='exercise' key={exercise[0]}>
                  <strong>{exercise[0]}</strong>
                  <span>{exercise[1]}</span>
                </div>
              ))}
            </div>
            <div className='form-row'>
              <button
                type='button'
                className='secondary'
                onClick={() => toggleWorkout(workout.id)}
              >
                <i className='fas fa-list' /> {t('Упражнения', 'Exercises')}
              </button>
              {completed ? (
                <button type='button' className='secondary' disabled>
                  <i className='fas fa-check' /> {t('Выполнено', 'Completed')}
                </button>
              ) : (
                canEdit && (
                  <button
                    type='button'
                    className='primary'
                    onClick={() =>
                      setActiveWorkout({
                        title,
                        exercises: workout.exercises,
                      })
                    }
                  >
                    <i className='fas fa-play' /> {t('Начать', 'Start')}
                  </button>
                )
              )}
            </div>
          </article>
        );
      })}

      {plans.length > 0 && (
        <>
          <div className='section'>
            <div className='between'>
              <h3>{t('Планы FitPulse', 'FitPulse plans')}</h3>
              <button
                type='button'
                className='secondary'
                onClick={() => generatePlan({ durationWeeks: 4 })}
              >
                {t('Сгенерировать', 'Generate')}
              </button>
            </div>
          </div>
          <section className='card'>
            {plans.map((plan, index) => (
              <div className='body-row' key={plan.id || index}>
                <span>{plan.title || plan.name || `#${index + 1}`}</span>
                <b>
                  {plan.duration_weeks || plan.durationWeeks || ''}{' '}
                  {t('нед.', 'wk')}
                </b>
              </div>
            ))}
          </section>
        </>
      )}

      <div className='section'>
        <h3>{t('Правила и гигиена', 'Rules and hygiene')}</h3>
      </div>
      <div className='grid-2'>
        <div className='fp-instructions' style={{ marginTop: 6 }}>
          <strong>
            <i
              className='fas fa-info-circle'
              style={{ color: 'var(--blue)' }}
            />{' '}
            {t('Правила в зале', 'Gym rules')}
          </strong>
          <p style={instructionTextStyle}>{instructions.gym}</p>
        </div>
        <div className='fp-instructions' style={{ marginTop: 6 }}>
          <strong>
            <i
              className='fas fa-info-circle'
              style={{ color: 'var(--blue)' }}
            />{' '}
            {t('Правила в бассейне', 'Pool rules')}
          </strong>
          <p style={instructionTextStyle}>{instructions.pool}</p>
        </div>
      </div>
      <div className='fp-instructions' style={{ marginTop: 6 }}>
        <strong>
          <i className='fas fa-info-circle' style={{ color: 'var(--blue)' }} />{' '}
          {t('Гигиена', 'Hygiene')}
        </strong>
        <p style={instructionTextStyle}>{instructions.hygiene}</p>
      </div>

      {videos.length > 0 && (
        <>
          <div className='section'>
            <h3>{t('Видео-уроки', 'Video lessons')}</h3>
          </div>
          {videos.map((video, index) => (
            <VideoCard
              key={video.id || index}
              video={video}
              index={index}
              onSelect={setVideoIndex}
            />
          ))}
        </>
      )}

      <div className='section'>
        <h3>{t('Добавить своё упражнение', 'Add your own exercise')}</h3>
      </div>
      {exercises.length > 0 && (
        <section className='card'>
          <div className='eyebrow'>{t('МОИ УПРАЖНЕНИЯ', 'MY EXERCISES')}</div>
          {exercises.map((exercise, index) => (
            <div className='body-row' key={`${exercise.name}-${index}`}>
              <strong>{exercise.name}</strong>
              <span>{exercise.sets}</span>
              <button
                type='button'
                className='icon'
                aria-label={t('Удалить упражнение', 'Delete exercise')}
                onClick={() => removeCustomExercise(index)}
              >
                <i className='fas fa-trash' />
              </button>
            </div>
          ))}
        </section>
      )}
      <section className='card form-card'>
        <label htmlFor='customExerciseName'>
          {t('Название упражнения', 'Exercise name')}
        </label>
        <input
          id='customExerciseName'
          value={exerciseName}
          onChange={(event) => setExerciseName(event.target.value)}
          placeholder={t(
            'Например, приседания с гантелей',
            'E.g. squats with dumbbells'
          )}
        />
        <label htmlFor='customExerciseSets'>
          {t('Подходы × Повторения', 'Sets × Reps')}
        </label>
        <input
          id='customExerciseSets'
          value={exerciseSets}
          onChange={(event) => setExerciseSets(event.target.value)}
          placeholder='4 × 12'
        />
        <div className='form-row'>
          {canEdit && (
            <button type='button' className='secondary' onClick={addExercise}>
              {t('Добавить', 'Add')}
            </button>
          )}
          <button
            type='button'
            className='primary'
            onClick={() => {
              quickAI(
                t('Предложи упражнение для дома', 'Suggest a home exercise')
              );
              go('ai');
            }}
          >
            <i className='fas fa-robot' /> {t('Спросить AI', 'Ask AI')}
          </button>
        </div>
      </section>

      {activeWorkout && (
        <WorkoutSession
          workout={activeWorkout}
          t={t}
          onClose={() => setActiveWorkout(null)}
          onFinish={() => startWorkout(activeWorkout.title)}
        />
      )}

      {videoIndex !== null && (
        <VideoModal
          videos={videos}
          index={videoIndex}
          onChange={handleVideoChange}
          onClose={() => setVideoIndex(null)}
        />
      )}
    </section>
  );
}

const instructionTextStyle = {
  marginTop: 4,
  color: 'var(--muted)',
  fontSize: 10,
  lineHeight: 1.5,
  whiteSpace: 'pre-line',
};
