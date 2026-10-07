import { useCallback } from 'react';

export function useNavigation({ state, update }) {
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
    },
    [state.screen, state.guest, state.legalReturnScreen, update]
  );

  const backFromLegal = useCallback(() => {
    const target = state.legalReturnScreen || (state.guest ? 'home' : 'login');
    update({ legalReturnScreen: null, screen: target });
  }, [state.legalReturnScreen, state.guest, update]);

  return { go, openLegal, backFromLegal };
}
