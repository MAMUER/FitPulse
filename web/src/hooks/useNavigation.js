import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';

export function useNavigation({ state, update }) {
  const navigate = useNavigate();

  const go = useCallback(
    (screen) => {
      if (
        state.screen === screen &&
        !state.selectedMeal &&
        !state.selectedMetric &&
        !state.showDateModal &&
        !state.selectedAchievement &&
        !state.selectedStory
      )
        return;
      void navigate(`/${screen}`);
      update({
        screen,
        selectedMeal: null,
        selectedMetric: null,
        showDateModal: false,
        selectedAchievement: null,
        selectedStory: null,
      });
    },
    [
      state.screen,
      state.selectedMeal,
      state.selectedMetric,
      state.showDateModal,
      state.selectedAchievement,
      state.selectedStory,
      update,
      navigate,
    ]
  );

  const openLegal = useCallback(
    (screen) => {
      const defaultReturn = state.guest ? 'home' : 'login';
      update({
        legalReturnScreen:
          screen === 'privacy' || screen === 'terms'
            ? state.legalReturnScreen || defaultReturn
            : state.screen,
        screen,
      });
      const path =
        screen === 'privacy' || screen === 'terms' ? screen : 'legal';
      void navigate(`/${path}`);
    },
    [state.screen, state.guest, state.legalReturnScreen, update, navigate]
  );

  const backFromLegal = useCallback(() => {
    // Если пришли из приложения (openLegal сохранил экран возврата) — возвращаемся
    // туда. Если зашли по прямой ссылке (legalReturnScreen пуст) — идём на
    // предыдущую страницу браузера (navigate(-1)), а не на home/login.
    if (state.legalReturnScreen) {
      const target = state.legalReturnScreen;
      update({ legalReturnScreen: null, screen: target });
      void navigate(`/${target}`);
      return;
    }
    if (typeof window !== 'undefined' && window.history.length > 1) {
      void navigate(-1);
      return;
    }
    const target = state.guest ? 'home' : 'login';
    update({ legalReturnScreen: null, screen: target });
    void navigate(`/${target}`);
  }, [state.legalReturnScreen, state.guest, update, navigate]);

  return { go, openLegal, backFromLegal };
}
