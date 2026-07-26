import { Chart } from 'chart.js/auto';
import { useEffect, useRef, useState } from 'react';
import {
  classifyState,
  getBiometricRecords,
  getPlan,
  getTrainingPlans,
} from '../../utils/api';
import { EXERCISE_NAME_MAP } from '../../utils/exerciseNames';
import './Dashboard.css';

export default function Dashboard() {
  const [hrValue, setHrValue] = useState('--');
  const [spo2Value, setSpo2Value] = useState('--');
  const [sleepValue, setSleepValue] = useState('--');
  const [bpValue, setBpValue] = useState('--/--');
  const [aiRecommendation, setAiRecommendation] = useState(
    'Загрузка рекомендаций...'
  );
  const [aiDescription, setAiDescription] = useState(
    'Анализируем ваши биометрические данные'
  );
  const [todayWorkout, setTodayWorkout] = useState('');
  const chartRef = useRef(null);
  const chartInstance = useRef(null);

  useEffect(() => {
    loadDashboard();
  }, [loadDashboard]);

  const loadDashboard = async () => {
    try {
      const [hrData, spo2Data, sleepData, systolicData, diastolicData] =
        await Promise.allSettled([
          getBiometricRecords('heart_rate', null, null, 10),
          getBiometricRecords('spo2', null, null, 5),
          getBiometricRecords('sleep_hours', null, null, 5),
          getBiometricRecords('systolic_pressure', null, null, 5),
          getBiometricRecords('diastolic_pressure', null, null, 5),
        ]);

      if (hrData.status === 'fulfilled' && hrData.value.records?.length > 0) {
        setHrValue(Math.round(hrData.value.records[0].value));
      }
      if (
        spo2Data.status === 'fulfilled' &&
        spo2Data.value.records?.length > 0
      ) {
        setSpo2Value(Math.round(spo2Data.value.records[0].value));
      }
      if (
        sleepData.status === 'fulfilled' &&
        sleepData.value.records?.length > 0
      ) {
        const sleepVal = sleepData.value.records[0].value;
        setSleepValue(
          Number.isInteger(sleepVal) ? sleepVal : sleepVal.toFixed(1)
        );
      }
      if (
        systolicData.status === 'fulfilled' &&
        systolicData.value.records?.length > 0 &&
        diastolicData.status === 'fulfilled' &&
        diastolicData.value.records?.length > 0
      ) {
        const sys = Math.round(systolicData.value.records[0].value);
        const dia = Math.round(diastolicData.value.records[0].value);
        setBpValue(`${sys}/${dia}`);
      }

      // Chart
      if (hrData.status === 'fulfilled' && hrData.value.records?.length > 1) {
        const records = hrData.value.records.slice(0, 20).reverse();
        const labels = records.map((r) =>
          new Date(r.timestamp).toLocaleTimeString('ru-RU', {
            hour: '2-digit',
            minute: '2-digit',
          })
        );
        const values = records.map((r) => r.value);

        if (chartInstance.current) chartInstance.current.destroy();
        const ctx = chartRef.current?.getContext('2d');
        if (ctx) {
          chartInstance.current = new Chart(ctx, {
            type: 'line',
            data: {
              labels,
              datasets: [
                {
                  data: values,
                  borderColor: '#ff375f',
                  backgroundColor: 'rgba(255,55,95,0.1)',
                  fill: true,
                  tension: 0.4,
                  pointRadius: 0,
                  borderWidth: 2.5,
                },
              ],
            },
            options: {
              responsive: true,
              maintainAspectRatio: false,
              plugins: { legend: { display: false } },
              scales: {
                x: {
                  display: true,
                  grid: { display: false },
                  ticks: {
                    color: '#636366',
                    maxTicksLimit: 6,
                    font: { size: 11 },
                  },
                },
                y: {
                  display: true,
                  grid: { color: 'rgba(255,255,255,0.05)' },
                  ticks: { color: '#636366', font: { size: 11 } },
                },
              },
            },
          });
        }
      }

      // AI recommendation
      try {
        const classifyRes = await classifyState({});
        if (classifyRes?.predicted_class_ru) {
          setAiRecommendation(classifyRes.predicted_class_ru);
          setAiDescription(classifyRes.description || '');
        } else if (classifyRes?.predicted_class) {
          setAiRecommendation(classifyRes.predicted_class);
          setAiDescription('AI анализ требует больше данных');
        }
      } catch {
        setAiRecommendation('Ошибка анализа');
        setAiDescription('Сервис AI временно недоступен');
      }

      // Today's workout
      try {
        const plansData = await getTrainingPlans(1, 1);
        const plans = plansData?.plans || [];
        if (plans.length > 0) {
          const plan = plans[0];
          let todayWorkoutHtml = '';
          try {
            const fullPlan = await getPlan(plan.plan_id);
            const planData =
              fullPlan?.plan?.plan_data || fullPlan?.plan_data || {};
            const weeks = planData.weeks || [];
            if (weeks.length > 0) {
              const today = new Date().getDay();
              let todayWorkoutData = null;
              for (const week of weeks) {
                for (const day of week.days || []) {
                  if (day.day_of_week === today) {
                    todayWorkoutData = day;
                    break;
                  }
                }
                if (todayWorkoutData) break;
              }
              if (todayWorkoutData) {
                const trainingTypes = {
                  cardio: '🏃 Кардио',
                  strength: '💪 Силовая',
                  recovery: '🧘 Восстановление',
                  endurance: '🏃 Выносливость',
                  hiit: 'HIIT',
                };
                const exercises = todayWorkoutData.exercises || [];
                const typeLabel =
                  trainingTypes[todayWorkoutData.training_type] || '';
                let exercisesHtml = '';
                if (exercises.length > 0) {
                  exercisesHtml =
                    '<ul style="margin: 10px 0; padding-left: 20px;">' +
                    exercises
                      .map((ex) => {
                        const details = [];
                        if (ex.sets) details.push(`${ex.sets}x${ex.reps}`);
                        if (ex.duration) details.push(`${ex.duration}мин`);
                        return `<li>${EXERCISE_NAME_MAP[ex.exercise_name] || ex.exercise_name || ''} ${details.length > 0 ? `(${details.join(', ')})` : ''}</li>`;
                      })
                      .join('') +
                    '</ul>';
                }
                todayWorkoutHtml = `
                  <div className="workout-content">
                    <h4>${typeLabel}</h4>
                    ${exercisesHtml}
                    ${todayWorkoutData.duration ? `<p> Длительность: ${todayWorkoutData.duration} мин</p>` : ''}
                    ${todayWorkoutData.notes ? `<p>${todayWorkoutData.notes}</p>` : ''}
                  </div>
                `;
              }
            }
          } catch (e) {
            console.warn('Could not load full plan details:', e);
          }
          if (!todayWorkoutHtml) {
            todayWorkoutHtml = `
              <div className="workout-content">
                <h4>😴 Отдых</h4>
                <p>Сегодня нет тренировки. Вашему организму нужен отдых для восстановления.</p>
              </div>
            `;
          }
          setTodayWorkout(todayWorkoutHtml);
        }
      } catch (err) {
        console.error('Failed to load today workout:', err);
      }
    } catch (err) {
      console.error('Dashboard load failed:', err);
    }
  };

  return (
    <div className='view active'>
      <section className='health-summary'>
        <div className='summary-card heart-rate'>
          <div className='card-icon'>❤️</div>
          <div className='card-data'>
            <span className='card-label'>Пульс</span>
            <span className='card-value' id='hrValue'>
              {hrValue}
            </span>
            <span className='card-unit'>уд/мин</span>
          </div>
        </div>
        <div className='summary-card spo2'>
          <div className='card-icon'>🫁</div>
          <div className='card-data'>
            <span className='card-label'>SpO₂</span>
            <span className='card-value' id='spo2Value'>
              {spo2Value}
            </span>
            <span className='card-unit'>%</span>
          </div>
        </div>
        <div className='summary-card sleep'>
          <div className='card-icon'>🌙</div>
          <div className='card-data'>
            <span className='card-label'>Сон</span>
            <span className='card-value' id='sleepValue'>
              {sleepValue}
            </span>
            <span className='card-unit'>часов</span>
          </div>
        </div>
        <div className='summary-card bp'>
          <div className='card-icon'>🩸</div>
          <div className='card-data'>
            <span className='card-label'>Давление</span>
            <span className='card-value' id='bpValue'>
              {bpValue}
            </span>
            <span className='card-unit'>мм рт.ст.</span>
          </div>
        </div>
      </section>

      <section className='chart-section'>
        <h3>Динамика пульса</h3>
        <div className='chart-container'>
          <canvas ref={chartRef} id='heartChart' />
        </div>
      </section>

      <section className='ai-section'>
        <div className='ai-card'>
          <div className='ai-header'>
            <span className='ai-badge'>AI Анализ</span>
          </div>
          <h3 id='aiRecommendation'>{aiRecommendation}</h3>
          <p id='aiDescription'>{aiDescription}</p>
        </div>
      </section>

      <section className='today-section'>
        <h3>🏋️ Тренировка на сегодня</h3>
        <div
          id='todayWorkout'
          className='workout-card'
          dangerouslySetInnerHTML={{
            __html:
              todayWorkout ||
              '<div className="workout-placeholder"><p>Сгенерируйте программу тренировок в разделе "Тренировки"</p></div>',
          }}
        />
      </section>
    </div>
  );
}
