import { useCallback } from 'react';
import * as api from '../services/api';
import { t } from '../utils/i18n';

export function useProfile({ state, update, notify }) {
  const editProfile = useCallback(async () => {
    const p = state.profile;
    const name = prompt(t('Имя', 'Name'), p.name);
    if (name === null) return;
    const status = prompt(
      t('Статус', 'Status'),
      p.status || t('Активен', 'Active')
    );
    if (status === null) return;
    const bio = prompt(t('О себе', 'About'), p.bio || '');
    if (bio === null) return;
    const phone = prompt(t('Телефон', 'Phone'), p.phone || '');
    if (phone === null) return;
    try {
      await api.saveProfile({
        full_name: name,
        gender: p.gender,
        phone,
        bio,
        status,
      });
      update({
        profile: {
          ...p,
          name: name.trim() || p.name,
          status: status.trim(),
          bio: bio.trim(),
          phone: phone.trim(),
        },
      });
      notify('Профиль обновлён');
    } catch {
      notify('Ошибка обновления профиля');
    }
  }, [state.profile, update, notify]);

  const changePassword = useCallback(() => {
    const oldPass = document.getElementById('oldPass')?.value || '';
    const np = document.getElementById('newPass')?.value || '';
    const np2 = document.getElementById('newPass2')?.value || '';
    try {
      api.changePassword(oldPass, np, np2);
    } catch (e) {
      notify(e.message);
      return;
    }
    update({ profile: { ...state.profile, password: np } });
    notify('Пароль изменён');
  }, [state.profile, update, notify]);

  const deleteAccount = useCallback(() => {
    if (
      !confirm(
        t(
          'Вы уверены, что хотите удалить аккаунт? Это действие необратимо.',
          'Are you sure you want to delete your account? This action is irreversible.'
        )
      )
    ) {
      return false;
    }
    api.deleteAccount().catch(() => {});
    notify(
      t(
        'Аккаунт удалён. Локальные данные очищены.',
        'Account deleted. Local data has been cleared.'
      )
    );
    update({ registered: false, guest: false, screen: 'login' });
    return true;
  }, [update, notify]);

  const searchFriends = useCallback(() => {
    const q = prompt(t('Имя или ID пользователя', 'User name or ID'), '');
    if (!q) return;
    const matches = (state.friends || []).filter(
      (f) =>
        String(f.name || '')
          .toLowerCase()
          .includes(q.toLowerCase()) ||
        String(f.id || '')
          .toLowerCase()
          .includes(q)
    );
    if (!matches.length) {
      notify(t('Ничего не найдено', 'Nothing found'));
      return;
    }
    const f = matches[0];
    f.added = !f.added;
    update({
      friends: [...(state.friends || [])],
      profile: {
        ...state.profile,
        friends: (state.friends || []).filter((x) => x.added).length,
      },
    });
  }, [state.friends, state.profile, update, notify]);

  const showDevices = useCallback(() => {
    const name = prompt(
      t(
        'Новое устройство (оставьте пустым для удаления)',
        'New device (leave empty to remove)'
      ),
      ''
    );
    if (name === null) return;
    const devices = [...(state.profile.devices || [])];
    if (name.trim()) devices.push(name.trim());
    else if (devices.length) devices.pop();
    update({ profile: { ...state.profile, devices } });
  }, [state.profile, update]);

  const loadFriends = useCallback(async () => {
    try {
      const data = await api.loadFriends();
      if (data?.friends) {
        update({ friends: data.friends });
      }
    } catch {
      // ignore — friends stay as-is (empty) when backend is unavailable
    }
  }, [update]);

  const addFriend = useCallback(
    async (name, status = 'offline') => {
      const trimmed = String(name || '').trim();
      if (!trimmed) {
        notify(t('Введите имя друга', 'Enter a friend name'));
        return false;
      }
      try {
        await api.addFriend(trimmed, '', status);
        notify(t('Друг добавлен', 'Friend added'));
        await loadFriends();
        return true;
      } catch {
        notify(t('Не удалось добавить друга', 'Failed to add friend'));
        return false;
      }
    },
    [notify, loadFriends]
  );

  const removeFriendById = useCallback(
    async (id) => {
      try {
        await api.removeFriend(id);
        notify(t('Друг удалён', 'Friend removed'));
        await loadFriends();
      } catch {
        notify(t('Не удалось удалить друга', 'Failed to remove friend'));
      }
    },
    [notify, loadFriends]
  );

  const loadLeaderboard = useCallback(async () => {
    try {
      const data = await api.loadLeaderboard();
      if (Array.isArray(data?.leaderboard)) {
        update({ leaderboard: data.leaderboard });
      }
    } catch {
      // ignore — leaderboard stays as-is when backend is unavailable
    }
  }, [update]);

  // syncLeaderboard отправляет на сервер текущий ник и очки пользователя.
  const syncLeaderboard = useCallback(async () => {
    try {
      await api.syncLeaderboard({
        display_name: state.profile?.name || '',
        points: state.points || 0,
        xp: state.xp || 0,
        workouts: state.dailyCompleted || 0,
      });
      await loadLeaderboard();
    } catch {
      // ignore
    }
  }, [
    state.profile,
    state.points,
    state.xp,
    state.dailyCompleted,
    loadLeaderboard,
  ]);

  const searchUsers = useCallback(
    async (q) => {
      const query = String(q || '').trim();
      if (query.length < 2) {
        update({ userSearchResults: [] });
        return;
      }
      try {
        const data = await api.searchUsers(query);
        update({
          userSearchResults: Array.isArray(data?.users) ? data.users : [],
        });
      } catch {
        update({ userSearchResults: [] });
      }
    },
    [update]
  );

  const sendFriendRequest = useCallback(
    async (userId) => {
      try {
        await api.sendFriendRequest(userId);
        notify(t('Заявка отправлена', 'Friend request sent'));
        return true;
      } catch {
        notify(t('Не удалось отправить заявку', 'Failed to send request'));
        return false;
      }
    },
    [notify]
  );

  const loadFriendRequests = useCallback(async () => {
    try {
      const data = await api.loadFriendRequests();
      if (Array.isArray(data?.requests)) {
        update({ friendRequests: data.requests });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const respondFriendRequest = useCallback(
    async (id, action) => {
      try {
        await api.respondFriendRequest(id, action);
        notify(
          action === 'accept'
            ? t('Заявка принята', 'Request accepted')
            : t('Заявка отклонена', 'Request declined')
        );
        await loadFriendRequests();
        await loadFriends();
      } catch {
        notify(t('Не удалось обработать заявку', 'Failed to handle request'));
      }
    },
    [notify, loadFriendRequests, loadFriends]
  );

  // reportUser отправляет жалобу на пользователя (напр. на фото) в oncall-bot.
  const reportUser = useCallback(
    async (userId) => {
      try {
        await api.reportUser(userId, 'Жалоба на фото');
        notify(t('Жалоба отправлена', 'Report sent'));
        return true;
      } catch {
        notify(t('Не удалось отправить жалобу', 'Failed to send report'));
        return false;
      }
    },
    [notify]
  );

  const loadProfile = useCallback(async () => {
    try {
      const data = await api.loadProfile();
      if (data?.profile) {
        const p = data.profile;
        update({
          profile: {
            ...state.profile,
            name: p.full_name || p.name || state.profile.name,
            email: p.email || state.profile.email,
            gender: p.gender || state.profile.gender,
            phone: p.phone || state.profile.phone,
            bio: p.bio || state.profile.bio,
            status: p.status || state.profile.status,
          },
          profileLoaded: true,
          authenticated: true,
        });
      } else {
        // No profile but the request succeeded → session is valid.
        update({ profileLoaded: true, authenticated: true });
      }
    } catch {
      // 401/403 or network error → not authenticated.
      update({ profileLoaded: true, authenticated: false });
    }
  }, [state.profile, update]);

  const saveProfile = useCallback(
    async (profileData) => {
      try {
        await api.saveProfile(profileData);
        notify('Профиль сохранён');
        return true;
      } catch {
        notify('Ошибка сохранения профиля');
        return false;
      }
    },
    [notify]
  );

  const uploadPhoto = useCallback(
    (event) => {
      const file = event.target.files?.[0];
      if (!file) return;
      if (file.size > 300_000) {
        notify(
          t(
            'Фото слишком большое (макс. 300 КБ)',
            'Photo too large (max 300 KB)'
          )
        );
        return;
      }
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result;
        update({ profileImage: typeof result === 'string' ? result : '' });
      };
      reader.readAsDataURL(file);
    },
    [update, notify]
  );

  const removePhoto = useCallback(() => {
    update({ profileImage: '' });
    notify(t('Фото удалено', 'Photo removed'));
  }, [update, notify]);

  const toggleFriendAdded = useCallback(
    (value) => {
      const index =
        typeof value === 'number'
          ? value
          : (state.friends || []).findIndex(
              (f) => f.name === value || String(f.id || '') === String(value)
            );
      const f = (state.friends || [])[index];
      if (!f) return;
      const newFriends = [...(state.friends || [])];
      newFriends[index] = { ...f, added: !f.added };
      update({
        friends: newFriends,
        profile: {
          ...state.profile,
          friends: newFriends.filter((x) => x.added).length,
        },
      });
    },
    [state.friends, state.profile, update]
  );

  const openFriendProfile = useCallback(
    (friendOrName) => {
      let friend;
      if (typeof friendOrName === 'string') {
        friend = (state.friends || []).find((f) => f.name === friendOrName);
      } else {
        friend = friendOrName;
      }
      if (!friend) {
        notify(t('Пользователь не найден', 'User not found'));
        return;
      }
      update({ selectedFriend: friend });
    },
    [state.friends, update, notify]
  );

  const showAchievement = useCallback(
    (name) => {
      const ach = (state.achievements || []).find((a) => a.name === name);
      if (ach) update({ selectedAchievement: ach });
    },
    [state.achievements, update]
  );

  const showAchievementsList = useCallback(() => {
    update({ selectedAchievement: null });
  }, [update]);

  return {
    editProfile,
    changePassword,
    deleteAccount,
    searchFriends,
    showDevices,
    loadProfile,
    loadFriends,
    addFriend,
    removeFriendById,
    loadLeaderboard,
    syncLeaderboard,
    searchUsers,
    sendFriendRequest,
    loadFriendRequests,
    respondFriendRequest,
    reportUser,
    saveProfile,
    uploadPhoto,
    removePhoto,
    toggleFriendAdded,
    openFriendProfile,
    showAchievement,
    showAchievementsList,
  };
}
