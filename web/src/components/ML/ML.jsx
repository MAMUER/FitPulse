import { Chart } from 'chart.js/auto';
import { useEffect, useRef, useState } from 'react';
import { classifyState, generateMLPlan, getPlan } from '../../utils/api';
import './ML.css';

const TRAINING_CLASSES = [
  { value: 'recovery', label: 'Восстановление' },
  { value: 'endurance_basic', label: 'Выносливость (базовая)' },
  { value: 'endurance_threshold', label: 'Выносливость (пороговая)' },
  { value: 'power_hiit', label: 'Силовая HIIT' },
  { value: 'overtraining', label: 'Перетренированность' },
  { value: 'illness', label: 'Болезнь' },
];

const DAY_NAMES = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'];

export default function ML() {
  const [classification, setClassification] = useState(null);
  const [classifying, setClassifying] = useState(false);
  const [plan, setPlan] = useState(null);
  const [generating, setGenerating] = useState(false);
  const [form, setForm] = useState({
    trainingClass: 'recovery',
    durationWeeks: 4,
    days: [1, 3, 5],
  });
  const chartRef = useRef(null);
  const chartInstance = useRef(null);

  const handleClassify = async () => {
    setClassifying(true);
    try {
      const data = await classifyState({});
      setClassification(data);
    } catch (e) {
      alert(`Ошибка анализа: ${e.message}`);
    } finally {
      setClassifying(false);
    }
  };

  const handleGeneratePlan = async (e) => {
    e.preventDefault();
    setGenerating(true);
    try {
      const userProfile = {};
      const goal = '';
      const constraints = {};
      const data = await generateMLPlan(
        form.trainingClass,
        userProfile,
        goal,
        constraints
      );
      setPlan(data);
    } catch (e) {
      alert(`Ошибка генерации: ${e.message}`);
    } finally {
      setGenerating(false);
    }
  };

  useEffect(() => {
    if (!plan) return;
    const renderPlanDetail = async () => {
      try {
        let planData = plan;
        if (plan.plan_id) {
          const full = await getPlan(plan.plan_id);
          planData = full?.plan || full;
        }
        const pd = planData?.plan_data || planData || {};
        const weeks = pd.weeks || [];
        if (weeks.length === 0) return;

        const progressCtx = chartRef.current?.getContext('2d');
        if (!progressCtx) return;
        if (chartInstance.current) chartInstance.current.destroy();

        const labels = [];
        const values = [];
        weeks.forEach((week, wi) => {
          (week.days || []).forEach((day) => {
            labels.push(`Нед ${wi + 1} ${DAY_NAMES[day.day_of_week] || ''}`);
            values.push(day.duration || 0);
          });
        });

        chartInstance.current = new Chart(progressCtx, {
          type: 'bar',
          data: {
            labels,
            datasets: [
              {
                label: 'Минуты',
                data: values,
                backgroundColor: 'rgba(255,55,95,0.6)',
                borderRadius: 8,
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
              y: {
                beginAtZero: true,
                ticks: { color: '#8e8e93' },
                grid: { color: 'rgba(255,255,255,0.05)' },
              },
              x: {
                ticks: {
                  color: '#8e8e93',
                  maxRotation: 45,
                  font: { size: 10 },
                },
                grid: { display: false },
              },
            },
          },
        });
      } catch (e) {
        console.error('Failed to render plan detail:', e);
      }
    };

    renderPlanDetail();
  }, [plan]);

  const renderPlanContent = () => {
    let planData = plan;
    if (plan?.plan_id && planData) {
      planData = plan;
    }
    const pd = planData?.plan_data || planData || {};
    const weeks = pd.weeks || [];

    if (weeks.length === 0) {
      return (
        <div className='empty-state'>
          <div className='empty-icon'>📋</div>
          <h3>План пуст</h3>
          <p>Попробуйте сгенерировать план с другими параметрами</p>
        </div>
      );
    }

    return weeks.map((week) => (
      <div key={week.week_number} className='plan-card'>
        <h4>Неделя {week.week_number}</h4>
        <div className='plan-meta' style={{ marginBottom: 10 }}>
          <span>
            Цель: {pd.goal || planData.training_goal || 'Общая тренировка'}
          </span>
        </div>
        {(week.days || []).map((day) => (
          <div
            key={day.day_id}
            style={{
              marginBottom: 12,
              paddingBottom: 12,
              borderBottom: '1px solid var(--bg-input)',
            }}
          >
            <div className='plan-day-header'>
              <div className='plan-day-name'>
                {DAY_NAMES[day.day_of_week] || `День ${day.day_of_week + 1}`}
              </div>
              <div className='plan-day-type'>{day.training_type || '—'}</div>
            </div>
            {(day.exercises || []).map((ex) => (
              <div key={ex.sort_order} className='exercise-item'>
                <div className='exercise-number'>{ex.sort_order + 1}</div>
                <div className='exercise-details'>
                  <div className='exercise-name'>
                    {ex.exercise_name || 'Упражнение'}
                  </div>
                  <div className='exercise-meta'>
                    {ex.sets && <span>{ex.sets} подходов</span>}
                    {ex.reps && <span> · {ex.reps} повторений</span>}
                    {ex.duration && <span> · {ex.duration} мин</span>}
                  </div>
                </div>
              </div>
            ))}
            {day.duration && (
              <div
                style={{
                  fontSize: 13,
                  color: 'var(--text-secondary)',
                  marginTop: 4,
                }}
              >
                Длительность: {day.duration} мин
              </div>
            )}
          </div>
        ))}
      </div>
    ));
  };

  return (
    <div className='view active'>
      <section className='ml-section'>
        <h3>Классификация состояния</h3>
        <button
          type='button'
          className='btn-primary'
          onClick={handleClassify}
          disabled={classifying}
          style={{ marginBottom: 16 }}
        >
          {classifying ? 'Анализ...' : 'Анализировать'}
        </button>
        {classification && (
          <div className='ml-result'>
            <div className='ml-classification'>
              <div className='class-label'>Состояние</div>
              <div className='class-name'>
                {classification.predicted_class_ru ||
                  classification.predicted_class ||
                  '—'}
              </div>
              <div className='confidence'>
                Уверенность:{' '}
                {classification.confidence
                  ? `${Math.round(classification.confidence * 100)}%`
                  : '—'}
              </div>
              {classification.description && (
                <p
                  style={{
                    marginTop: 8,
                    color: 'var(--text-secondary)',
                    fontSize: 14,
                  }}
                >
                  {classification.description}
                </p>
              )}
            </div>
          </div>
        )}
      </section>

      <section className='ml-section'>
        <h3>Генерация плана</h3>
        <form className='ml-form' onSubmit={handleGeneratePlan}>
          <div className='form-group'>
            <label htmlFor='training-class'>Тип тренировки</label>
            <select
              id='training-class'
              value={form.trainingClass}
              onChange={(e) =>
                setForm((f) => ({ ...f, trainingClass: e.target.value }))
              }
            >
              {TRAINING_CLASSES.map((tc) => (
                <option key={tc.value} value={tc.value}>
                  {tc.label}
                </option>
              ))}
            </select>
          </div>
          <div className='form-group'>
            <label htmlFor='duration-weeks'>Длительность (недель)</label>
            <input
              id='duration-weeks'
              type='number'
              min='1'
              max='12'
              value={form.durationWeeks}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  durationWeeks: Number(e.target.value),
                }))
              }
            />
          </div>
          <div className='form-group'>
            <label htmlFor='training-days'>Дни тренировок</label>
            <div id='training-days' className='days-grid'>
              {DAY_NAMES.map((d, idx) => (
                <label
                  key={idx}
                  className={`day-chip ${form.days.includes(idx) ? 'selected' : ''}`}
                >
                  <input
                    type='checkbox'
                    checked={form.days.includes(idx)}
                    onChange={() =>
                      setForm((f) => ({
                        days: f.days.includes(idx)
                          ? f.days.filter((d) => d !== idx)
                          : [...f.days, idx],
                      }))
                    }
                  />
                  {d}
                </label>
              ))}
            </div>
          </div>
          <button type='submit' className='btn-primary' disabled={generating}>
            {generating ? 'Генерация...' : 'Сгенерировать план'}
          </button>
        </form>
      </section>

      {plan && (
        <section>
          <h3 style={{ marginBottom: 12 }}>Сгенерированный план</h3>
          <div id='generatedPlanDetail' className='plans-list'>
            {renderPlanContent()}
          </div>
          <div
            style={{
              background: 'var(--bg-card)',
              borderRadius: 'var(--radius-lg)',
              padding: 16,
              marginTop: 16,
            }}
          >
            <h4
              style={{
                marginBottom: 12,
                fontSize: 14,
                color: 'var(--text-secondary)',
              }}
            >
              Прогресс
            </h4>
            <div style={{ position: 'relative', height: 220 }}>
              <canvas ref={chartRef} id='mlProgressChart' />
            </div>
          </div>
        </section>
      )}
    </div>
  );
}
