import { useCallback } from 'react';
import * as api from '../services/api';
import { t } from '../utils/i18n';

export function useSurvey({ state, update, notify }) {
  const surveySelect = useCallback(
    (key, value, isMulti) => {
      const survey = { ...state.survey };
      if (isMulti) {
        const arr = [...(survey[key] || [])];
        const idx = arr.indexOf(value);
        if (idx > -1) arr.splice(idx, 1);
        else arr.push(value);
        survey[key] = arr;
      } else {
        survey[key] = value;
      }
      update({ survey, surveyDeferred: false });
    },
    [state.survey, update]
  );

  const saveSurveyToBackend = useCallback(async (survey) => {
    try {
      await api.saveSurvey({ survey, survey_completed: true });
    } catch {
      // ignore save errors, localStorage fallback
    }
  }, []);

  const loadSurveyFromBackend = useCallback(async () => {
    try {
      const data = await api.loadSurvey();
      if (data?.survey) {
        update({ survey: data.survey, surveyCompleted: data.survey_completed });
      }
    } catch {
      // ignore load errors
    }
  }, [update]);

  const surveyNext = useCallback(async () => {
    const steps = [
      'allergies',
      'restrictions',
      'goals',
      'diet',
      'activity',
      'sleepHours',
      'waterIntake',
      'preferredWorkouts',
      'trainingDays',
    ];
    const key = steps[state.survey.step];
    const val = state.survey[key];
    if (Array.isArray(val) && val.length === 0) {
      notify(t('Выберите хотя бы один вариант', 'Select at least one option'));
      return;
    }
    if (typeof val === 'string' && !val) {
      notify(t('Выберите один вариант', 'Select one option'));
      return;
    }
    if (state.survey.step < steps.length - 1) {
      update({ survey: { ...state.survey, step: state.survey.step + 1 } });
    } else {
      update({ surveyCompleted: true, surveyDeferred: false, screen: 'home' });
      await saveSurveyToBackend(state.survey);
    }
  }, [state.survey, update, notify, saveSurveyToBackend]);

  const surveyPrev = useCallback(() => {
    if (state.survey.step > 0) {
      update({ survey: { ...state.survey, step: state.survey.step - 1 } });
    }
  }, [state.survey, update]);

  const deferSurvey = useCallback(() => {
    update({ surveyDeferred: true, screen: 'home' });
  }, [update]);

  const resumeSurvey = useCallback(() => {
    update({ screen: 'survey', surveyDeferred: false });
  }, [update]);

  return {
    surveySelect,
    surveyNext,
    surveyPrev,
    deferSurvey,
    resumeSurvey,
    loadSurveyFromBackend,
  };
}
