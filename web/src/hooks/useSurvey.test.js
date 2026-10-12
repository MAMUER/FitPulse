import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../services/api';
import { useSurvey } from './useSurvey';

vi.mock('../services/api', () => ({
  saveSurvey: vi.fn(),
  loadSurvey: vi.fn(),
}));

function makeState(overrides = {}) {
  return {
    survey: {
      step: 0,
      allergies: [],
      restrictions: [],
      goals: [],
      diet: '',
      activity: '',
      sleepHours: '',
      waterIntake: '',
      preferredWorkouts: [],
      trainingDays: [],
    },
    ...overrides,
  };
}

let state;
let update;
let notify;
let result;

function setup(overrides = {}) {
  state = makeState(overrides);
  update = vi.fn();
  notify = vi.fn();
  result = renderHook(() => useSurvey({ state, update, notify })).result;
}

beforeEach(() => {
  vi.resetAllMocks();
});

describe('useSurvey', () => {
  it('returns all survey callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'deferSurvey',
      'loadSurveyFromBackend',
      'resumeSurvey',
      'surveyNext',
      'surveyPrev',
      'surveySelect',
    ]);
  });

  it('surveySelect stores single value', () => {
    setup();
    result.current.surveySelect('diet', 'vegan', false);
    expect(update).toHaveBeenCalledWith({
      survey: expect.objectContaining({ step: 0, diet: 'vegan' }),
      surveyDeferred: false,
    });
  });

  it('surveySelect appends multi value', () => {
    setup({ survey: { ...makeState().survey, allergies: ['глютен'] } });
    result.current.surveySelect('allergies', 'лактоза', true);
    expect(update).toHaveBeenCalledWith({
      survey: expect.objectContaining({ allergies: ['глютен', 'лактоза'] }),
      surveyDeferred: false,
    });
  });

  it('surveySelect removes existing multi value', () => {
    setup({
      survey: { ...makeState().survey, allergies: ['глютен', 'лактоза'] },
    });
    result.current.surveySelect('allergies', 'глютен', true);
    expect(update).toHaveBeenCalledWith({
      survey: expect.objectContaining({ allergies: ['лактоза'] }),
      surveyDeferred: false,
    });
  });

  it('loadSurveyFromBackend restores answers', async () => {
    setup();
    api.loadSurvey.mockResolvedValue({
      survey: { ...makeState().survey, diet: 'vegan' },
      survey_completed: true,
    });
    await act(async () => {
      await result.current.loadSurveyFromBackend();
    });
    expect(update).toHaveBeenCalledWith({
      survey: expect.objectContaining({ diet: 'vegan' }),
      surveyCompleted: true,
    });
  });

  it('loadSurveyFromBackend ignores empty payload', async () => {
    setup();
    api.loadSurvey.mockResolvedValue({});
    await act(async () => {
      await result.current.loadSurveyFromBackend();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('loadSurveyFromBackend swallows errors', async () => {
    setup();
    api.loadSurvey.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.loadSurveyFromBackend();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('surveyNext requires at least one option', async () => {
    setup();
    await act(async () => {
      await result.current.surveyNext();
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Выберите хотя бы один вариант');
  });

  it('surveyNext requires a single option', async () => {
    setup({ survey: { ...makeState().survey, step: 3 } });
    await act(async () => {
      await result.current.surveyNext();
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Выберите один вариант');
  });

  it('surveyNext advances to next step', async () => {
    setup({
      survey: {
        ...makeState().survey,
        allergies: ['глютен'],
        restrictions: ['мясо'],
      },
    });
    await act(async () => {
      await result.current.surveyNext();
    });
    expect(update).toHaveBeenCalledWith({
      survey: expect.objectContaining({ step: 1 }),
    });
    expect(api.saveSurvey).not.toHaveBeenCalled();
  });

  it('surveyNext completes and saves on last step', async () => {
    setup({
      survey: {
        ...makeState().survey,
        step: 8,
        trainingDays: ['mon', 'wed'],
      },
    });
    api.saveSurvey.mockResolvedValue({});
    await act(async () => {
      await result.current.surveyNext();
    });
    expect(update).toHaveBeenCalledWith({
      surveyCompleted: true,
      surveyDeferred: false,
      screen: 'home',
    });
    expect(api.saveSurvey).toHaveBeenCalledWith({
      survey: expect.objectContaining({
        step: 8,
        trainingDays: ['mon', 'wed'],
      }),
      survey_completed: true,
    });
  });

  it('surveyNext ignores save errors', async () => {
    setup({
      survey: { ...makeState().survey, step: 8, trainingDays: ['mon'] },
    });
    api.saveSurvey.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.surveyNext();
    });
    expect(update).toHaveBeenCalledWith({
      surveyCompleted: true,
      surveyDeferred: false,
      screen: 'home',
    });
  });

  it('surveyPrev goes back a step', () => {
    setup({ survey: { ...makeState().survey, step: 3 } });
    result.current.surveyPrev();
    expect(update).toHaveBeenCalledWith({
      survey: expect.objectContaining({ step: 2 }),
    });
  });

  it('surveyPrev stops at first step', () => {
    setup();
    result.current.surveyPrev();
    expect(update).not.toHaveBeenCalled();
  });

  it('deferSurvey hides survey', () => {
    setup();
    result.current.deferSurvey();
    expect(update).toHaveBeenCalledWith({
      surveyDeferred: true,
      screen: 'home',
    });
  });

  it('resumeSurvey shows survey again', () => {
    setup();
    result.current.resumeSurvey();
    expect(update).toHaveBeenCalledWith({
      screen: 'survey',
      surveyDeferred: false,
    });
  });
});
