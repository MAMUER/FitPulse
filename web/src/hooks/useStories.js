import { useCallback } from 'react';

export function useStories({ state, update }) {
  const scrollStories = useCallback((_dir) => {
    // Handled by StoryCarousel component
  }, []);

  const showAllStories = useCallback(() => {
    if (!state.stories?.length) return;
    update({ selectedStory: state.stories[0] });
  }, [state.stories, update]);

  const openStory = useCallback(
    (index) => {
      if (!state.stories[index]) return;
      update({
        selectedStory: state.stories[index],
        storySeen: { ...state.storySeen, [index]: true },
      });
    },
    [state.stories, state.storySeen, update]
  );

  const closeStory = useCallback(() => {
    update({ selectedStory: null, storyPaused: false });
  }, [update]);

  const nextStory = useCallback(() => {
    const i = state.stories.indexOf(state.selectedStory);
    if (i < state.stories.length - 1) {
      update({
        selectedStory: state.stories[i + 1],
        storySeen: { ...state.storySeen, [i + 1]: true },
      });
    } else {
      closeStory();
    }
  }, [state.stories, state.selectedStory, state.storySeen, update, closeStory]);

  const prevStory = useCallback(() => {
    const i = state.stories.indexOf(state.selectedStory);
    if (i > 0) {
      update({
        selectedStory: state.stories[i - 1],
        storySeen: { ...state.storySeen, [i - 1]: true },
      });
    }
  }, [state.stories, state.selectedStory, state.storySeen, update]);

  return {
    scrollStories,
    showAllStories,
    openStory,
    closeStory,
    nextStory,
    prevStory,
  };
}
