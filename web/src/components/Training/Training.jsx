import { useEffect, useState } from 'react';
import {
  generateTrainingPlan as apiGeneratePlan,
  classifyState,
  getTrainingPlans,
} from '../../utils/api';
import './Training.css';

export default function Training() {
  const [plans, setPlans] = useState([]);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);

  useEffect(() => {
    loadPlans();
  }, [loadPlans]);

  const loadPlans = async () => {
    try {
      const data = await getTrainingPlans();
      setPlans(data?.plans || []);
    } catch (e) {
      console.error('Failed to load plans:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleGenerate = async () => {
    setGenerating(true);
    try {
      let trainingClass = 'recovery';
      let confidence = 0.5;
      try {
        const classifyRes = await classifyState({});
        trainingClass = classifyRes.predicted_class || 'recovery';
        confidence = classifyRes.confidence || 0.5;
      } catch {
        // use defaults
      }
      await apiGeneratePlan(4, [1, 3, 5], trainingClass, confidence);
      await loadPlans();
    } catch (err) {
      console.error('Failed to generate plan:', err);
      alert(`Ошибка генерации: ${err.message}`);
    } finally {
      setGenerating(false);
    }
  };

  if (loading) return <div className='loading'>Загрузка программ...</div>;

  return (
    <div className='view active'>
      <div id='plansList' className='plans-list'>
        {plans.length === 0 ? (
          <div className='empty-state'>
            <div className='empty-icon'>🏃</div>
            <h3>Нет активных программ</h3>
            <p>AI создаст персональный план на основе ваших данных</p>
          </div>
        ) : (
          plans.map((plan) => (
            <div key={plan.plan_id} className='plan-card'>
              <h4>{plan.plan_data?.name || 'Персонализированная программа'}</h4>
              <div className='plan-meta'>
                <span>Цель: {plan.training_goal || 'Общая тренировка'}</span>
                <span>{plan.duration_weeks || 4} недель</span>
              </div>
            </div>
          ))
        )}
      </div>
      <button
        id='generatePlanBtn'
        className='btn-floating'
        onClick={handleGenerate}
        disabled={generating}
        aria-label='Сгенерировать план'
      >
        {generating ? (
          '...'
        ) : (
          <svg
            width='28'
            height='28'
            viewBox='0 0 24 24'
            fill='none'
            stroke='currentColor'
            strokeWidth='2.5'
          >
            <line x1='12' y1='5' x2='12' y2='19' />
            <line x1='5' y1='12' x2='19' y2='12' />
          </svg>
        )}
      </button>
    </div>
  );
}
