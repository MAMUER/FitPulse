import { useCallback } from 'react';
import { t } from '../utils/i18n';
import * as api from '../services/api';

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
  }, [state.profile, update, notify, t]);

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
    )
      return;
    api.deleteAccount();
    notify(
      t(
        'Аккаунт удалён. Локальные данные очищены.',
        'Account deleted. Local data has been cleared.'
      )
    );
    update({ registered: false, guest: false, screen: 'login' });
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
        });
      } else {
        update({ profileLoaded: true });
      }
    } catch {
      update({ profileLoaded: true });
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
      const reader = new FileReader();
      reader.onload = () => {
        update({ profileImage: reader.result });
      };
      reader.readAsDataURL(file);
    },
    [update]
  );

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
    saveProfile,
    uploadPhoto,
    toggleFriendAdded,
    openFriendProfile,
    showAchievement,
    showAchievementsList,
  };
}
