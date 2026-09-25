import { useCallback } from 'react';
import { t } from '../utils/i18n';

export function useMedia({ state, update, notify }) {
  const addCustomExercise = useCallback(() => {
    const name = document.getElementById('customExerciseName')?.value.trim();
    const sets = document.getElementById('customExerciseSets')?.value.trim();
    if (!name || !sets) {
      notify(t('Заполните все поля', 'Please fill all fields'));
      return;
    }
    update({
      customExercises: [...(state.customExercises || []), { name, sets }],
    });
    if (document.getElementById('customExerciseName'))
      document.getElementById('customExerciseName').value = '';
    if (document.getElementById('customExerciseSets'))
      document.getElementById('customExerciseSets').value = '';
  }, [state.customExercises, update, notify]);

  const removeCustomExercise = useCallback(
    (index) => {
      if (!confirm(t('Удалить упражнение?', 'Delete exercise?'))) return;
      update({
        customExercises: (state.customExercises || []).filter(
          (_, i) => i !== index
        ),
      });
    },
    [state.customExercises, update]
  );

  const openVideoModal = useCallback(
    (id) => {
      const video = state.trainingVideos.find((v) => v.id === id);
      if (!video) return;
      update({
        videoIndex: state.trainingVideos.indexOf(video),
        selectedVideoId: id,
      });
    },
    [state.trainingVideos, update]
  );

  const closeVideoModal = useCallback(() => {
    update({ selectedVideoId: null });
  }, [update]);

  const changeVideoModal = useCallback(
    (direction) => {
      const total = state.trainingVideos.length;
      const newIndex = (state.videoIndex + direction + total) % total;
      update({
        videoIndex: newIndex,
        selectedVideoId: state.trainingVideos[newIndex]?.id,
      });
    },
    [state.trainingVideos, state.videoIndex, update]
  );

  const connectVideo = useCallback(() => {
    const video = state.trainingVideos.find(
      (v) => v.id === state.selectedVideoId
    );
    if (video) {
      notify(
        `${video.title}\n\n${t('Это демонстрационный видео-урок. Реальный видеосервис подключается через backend/API.', 'This is a demo video lesson. A real video service is connected through the backend/API.')}`
      );
    }
  }, [state.trainingVideos, state.selectedVideoId, notify]);

  return {
    addCustomExercise,
    removeCustomExercise,
    openVideoModal,
    closeVideoModal,
    changeVideoModal,
    connectVideo,
  };
}
