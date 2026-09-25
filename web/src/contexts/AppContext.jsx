import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { getInitialState } from '../reducers/appState';
import { saveState } from '../services/storage';
import {
  formatRelativeDate,
  safeText,
} from '../utils/helpers';
import { getMonthName, t } from '../utils/i18n';
import { useAuth } from '../hooks/useAuth';
import { useNavigation } from '../hooks/useNavigation';
import { useUI } from '../hooks/useUI';
import { useCalendar } from '../hooks/useCalendar';
import { useNutrition } from '../hooks/useNutrition';
import { useTraining } from '../hooks/useTraining';
import { useHealth } from '../hooks/useHealth';
import { useChat } from '../hooks/useChat';
import { useProfile } from '../hooks/useProfile';
import { useStories } from '../hooks/useStories';
import { useSurvey } from '../hooks/useSurvey';
import { useAI } from '../hooks/useAI';
import { useMedia } from '../hooks/useMedia';
import { useSettings } from '../hooks/useSettings';

const AppContext = createContext(null);

export function AppProvider({ children }) {
  const [state, setState] = useState(() => getInitialState());

  const update = useCallback((updates) => {
    setState((prev) => {
      if (typeof updates === 'function') {
        updates = updates(prev);
      }
      return { ...prev, ...updates };
    });
  }, []);

  const notify = useCallback(
    (message, duration = 2600) => {
      update({ __toast: message, __toastTimer: Date.now() });
      setTimeout(() => update({ __toast: null }), duration);
    },
    [update]
  );

  const save = useCallback(() => {
    try {
      saveState({
        theme: state.theme,
        highContrast: state.highContrast,
        language: state.language,
        profileImage: state.profileImage,
        profileName: state.profile.name,
        gender: state.profile.gender,
        email: state.profile.email,
        phone: state.profile.phone,
        twoFactor: state.profile.twoFactor,
        assistantStyle: state.profile.assistantStyle,
        devices: state.profile.devices,
        bio: state.profile.bio,
        status: state.profile.status,
        friendsCount: state.profile.friends,
        calendarEvents: state.calendarEvents,
        customExercises: state.customExercises,
        customMeals: state.customMeals,
        waterIntake: state.waterIntake,
        waterDate: state.waterDate,
        points: state.points,
        level: state.level,
        streak: state.streak,
        maxStreak: state.maxStreak,
        friends: state.friends,
        pinnedPlaces: state.pinnedPlaces,
        hackIndex: state.hackIndex,
        nutritionPeriod: state.nutritionPeriod,
        trainingPeriod: state.trainingPeriod,
        calendarMonthIndex: state.calendarMonthIndex,
        calendarYear: state.calendarYear,
        completedWorkouts: state.completedWorkouts,
        dailyCompleted: state.dailyCompleted,
        xp: state.xp,
        trainingMinutes: state.trainingMinutes,
        caloriesBurned: state.caloriesBurned,
        biometrics: state.biometrics,
        bodyComposition: state.bodyComposition,
        trainingPlans: state.trainingPlans,
        selectedPlanId: state.selectedPlanId,
        mealsList: state.mealsList,
        calendarEventsList: state.calendarEventsList,
        achievementsBackend: state.achievementsBackend,
        videos: state.videos,
        conditions: state.conditions,
        menstrualCycles: state.menstrualCycles,
        aiClassification: state.aiClassification,
        aiPlan: state.aiPlan,
        aiDiet: state.aiDiet,
        survey: state.survey,
        surveyCompleted: state.surveyCompleted,
        surveyDeferred: state.surveyDeferred,
        registered: state.registered,
        guest: state.guest,
        region: state.region,
        chatConsent: state.chatConsent,
        chatSettings: state.chatSettings,
        chatMessages: state.chatMessages,
        storySeen: state.storySeen,
        twoFactorSetup: state.twoFactorSetup,
        twoFactorTempToken: state.twoFactorTempToken,
      });
    } catch {
      // localStorage full or unavailable
    }
  }, [state]);

  useEffect(() => {
    save();
  }, [save]);

  useEffect(() => {
    document.documentElement.dataset.theme = state.theme;
    document.documentElement.dataset.highcontrast = state.highContrast
      ? 'on'
      : 'off';
    document.documentElement.lang = state.language === 'en' ? 'en' : 'ru';
  }, [state.theme, state.language, state.highContrast]);

  const auth = useAuth({ state, update, notify });
  const navigation = useNavigation({ state, update });
  const ui = useUI({ state, update, notify });
  const calendar = useCalendar({
    state,
    update,
    notify,
    getMonthName,
    language: state.language,
  });
  const nutrition = useNutrition({
    state,
    update,
    notify,
    getMonthName,
    language: state.language,
  });
  const training = useTraining({ state, update, notify });
  const health = useHealth({ state, update, notify, t });
  const chat = useChat({ state, update, notify });
  const profile = useProfile({ state, update, notify });
  const stories = useStories({ state, update });
  const survey = useSurvey({ state, update, notify });
  const ai = useAI({ state, update, notify });
  const media = useMedia({ state, update, notify });
  const settings = useSettings({ state, update, notify });

  useEffect(() => {
    if (state.registered && !state.profileLoaded) {
      profile.loadProfile();
      health.loadBiometrics();
      training.loadTrainingPlans();
      health.loadConditions();
      health.loadMenstrualCycles();
      health.loadBodyComposition();
    }
  }, [
    state.registered,
    state.profileLoaded,
    profile.loadProfile,
    health.loadBiometrics,
    training.loadTrainingPlans,
    health.loadConditions,
    health.loadMenstrualCycles,
    health.loadBodyComposition,
  ]);

  const value = useMemo(
    () => ({
      state,
      t,
      getMonthName,
      safeText,
      formatRelativeDate,
      notify,
      ...auth,
      ...navigation,
      ...ui,
      ...calendar,
      ...nutrition,
      ...training,
      ...health,
      ...chat,
      ...profile,
      ...stories,
      ...survey,
      ...ai,
      ...media,
      ...settings,
    }),
    [
      state,
      t,
      notify,
      auth,
      navigation,
      ui,
      calendar,
      nutrition,
      training,
      health,
      chat,
      profile,
      stories,
      survey,
      ai,
      media,
      settings,
    ]
  );

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp() {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error('useApp must be used within AppProvider');
  return ctx;
}
