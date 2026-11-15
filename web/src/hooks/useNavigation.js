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
        selectedChat: screen === 'chat' ? null : state.selectedChat,
      });
    },
    [
      state.screen,
      state.selectedMeal,
      state.selectedMetric,
      state.showDateModal,
      state.selectedAchievement,
      state.selectedStory,
      state.selectedChat,
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
    const target = state.legalReturnScreen || (state.guest ? 'home' : 'login');
    update({ legalReturnScreen: null, screen: target });
    void navigate(`/${target}`);
  }, [state.legalReturnScreen, state.guest, update, navigate]);

  return { go, openLegal, backFromLegal };
}
