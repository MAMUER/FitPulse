import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../services/api';
import { useProfile } from './useProfile';

vi.mock('../services/api', () => ({
  saveProfile: vi.fn(),
  changePassword: vi.fn(),
  deleteAccount: vi.fn(),
  loadProfile: vi.fn(),
}));

function makeState(overrides = {}) {
  return {
    profile: {
      name: 'Михаил',
      email: 'mih@example.com',
      gender: 'male',
      status: 'Активен',
      bio: '',
      phone: '',
      devices: [],
    },
    friends: [],
    achievements: [],
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
  result = renderHook(() => useProfile({ state, update, notify })).result;
}

function setValue(id, value) {
  const el = document.getElementById(id);
  if (el) el.value = value;
}

beforeEach(() => {
  vi.resetAllMocks();
  for (const id of ['oldPass', 'newPass', 'newPass2']) {
    const el = document.createElement('input');
    el.id = id;
    document.body.appendChild(el);
  }
  api.deleteAccount.mockResolvedValue(true);
});

describe('useProfile', () => {
  it('returns all profile callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'changePassword',
      'deleteAccount',
      'editProfile',
      'loadProfile',
      'openFriendProfile',
      'saveProfile',
      'searchFriends',
      'showAchievement',
      'showAchievementsList',
      'showDevices',
      'toggleFriendAdded',
      'uploadPhoto',
    ]);
  });

  it('editProfile stops when name prompt is cancelled', async () => {
    setup();
    window.prompt = vi.fn().mockReturnValue(null);
    await act(async () => {
      await result.current.editProfile();
    });
    expect(api.saveProfile).not.toHaveBeenCalled();
    expect(update).not.toHaveBeenCalled();
  });

  it('editProfile stops when status prompt is cancelled', async () => {
    setup();
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Новое имя')
      .mockReturnValueOnce(null);
    await act(async () => {
      await result.current.editProfile();
    });
    expect(api.saveProfile).not.toHaveBeenCalled();
  });

  it('editProfile stops when bio prompt is cancelled', async () => {
    setup();
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Новое имя')
      .mockReturnValueOnce('Новый статус')
      .mockReturnValueOnce(null);
    await act(async () => {
      await result.current.editProfile();
    });
    expect(api.saveProfile).not.toHaveBeenCalled();
  });

  it('editProfile stops when phone prompt is cancelled', async () => {
    setup();
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('Новое имя')
      .mockReturnValueOnce('Новый статус')
      .mockReturnValueOnce('О себе')
      .mockReturnValueOnce(null);
    await act(async () => {
      await result.current.editProfile();
    });
    expect(api.saveProfile).not.toHaveBeenCalled();
  });

  it('editProfile saves profile', async () => {
    setup({ profile: { ...makeState().profile, bio: 'Бегун', phone: '123' } });
    window.prompt = vi
      .fn()
      .mockReturnValueOnce(' Новое имя ')
      .mockReturnValueOnce(' Новый статус ')
      .mockReturnValueOnce(' Новое обо мне ')
      .mockReturnValueOnce(' 999 ');
    api.saveProfile.mockResolvedValue({});
    await act(async () => {
      await result.current.editProfile();
    });
    expect(api.saveProfile).toHaveBeenCalledWith({
      full_name: ' Новое имя ',
      gender: 'male',
      phone: ' 999 ',
      bio: ' Новое обо мне ',
      status: ' Новый статус ',
    });
    expect(update).toHaveBeenCalledWith({
      profile: {
        name: 'Новое имя',
        email: 'mih@example.com',
        gender: 'male',
        status: 'Новый статус',
        bio: 'Новое обо мне',
        phone: '999',
        devices: [],
      },
    });
    expect(notify).toHaveBeenCalledWith('Профиль обновлён');
  });

  it('editProfile keeps previous name when blank', async () => {
    setup();
    window.prompt = vi
      .fn()
      .mockReturnValueOnce('   ')
      .mockReturnValueOnce('   ')
      .mockReturnValueOnce('')
      .mockReturnValueOnce('');
    api.saveProfile.mockResolvedValue({});
    await act(async () => {
      await result.current.editProfile();
    });
    expect(update).toHaveBeenCalledWith({
      profile: expect.objectContaining({
        name: 'Михаил',
        status: '',
        bio: '',
        phone: '',
      }),
    });
  });

  it('editProfile notifies on failure', async () => {
    setup();
    window.prompt = vi.fn().mockReturnValue('value');
    api.saveProfile.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.editProfile();
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Ошибка обновления профиля');
  });

  it('changePassword updates profile', () => {
    setup();
    setValue('oldPass', 'oldpass');
    setValue('newPass', 'newpass');
    setValue('newPass2', 'newpass');
    result.current.changePassword();
    expect(api.changePassword).toHaveBeenCalledWith(
      'oldpass',
      'newpass',
      'newpass'
    );
    expect(update).toHaveBeenCalledWith({
      profile: { ...state.profile, password: 'newpass' },
    });
    expect(notify).toHaveBeenCalledWith('Пароль изменён');
  });

  it('changePassword surfaces validation errors', () => {
    setup();
    api.changePassword.mockImplementation(() => {
      throw new Error('Пароли не совпадают');
    });
    result.current.changePassword();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Пароли не совпадают');
  });

  it('deleteAccount clears data when confirmed', () => {
    setup();
    window.confirm = vi.fn().mockReturnValue(true);
    result.current.deleteAccount();
    expect(api.deleteAccount).toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({
      registered: false,
      guest: false,
      screen: 'login',
    });
    expect(notify).toHaveBeenCalledWith(
      'Аккаунт удалён. Локальные данные очищены.'
    );
  });

  it('deleteAccount does nothing when cancelled', () => {
    setup();
    window.confirm = vi.fn().mockReturnValue(false);
    result.current.deleteAccount();
    expect(api.deleteAccount).not.toHaveBeenCalled();
    expect(update).not.toHaveBeenCalled();
  });

  it('deleteAccount continues when backend call fails', () => {
    setup();
    window.confirm = vi.fn().mockReturnValue(true);
    api.deleteAccount.mockRejectedValue(new Error('boom'));
    expect(() => result.current.deleteAccount()).not.toThrow();
    expect(update).toHaveBeenCalledWith({
      registered: false,
      guest: false,
      screen: 'login',
    });
    expect(notify).toHaveBeenCalledWith(
      'Аккаунт удалён. Локальные данные очищены.'
    );
  });

  it('searchFriends notifies when nothing found', () => {
    setup({ friends: [{ name: 'Анна', id: '1', added: false }] });
    window.prompt = vi.fn().mockReturnValue('zzz');
    result.current.searchFriends();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Ничего не найдено');
  });

  it('searchFriends stops on empty query', () => {
    setup({ friends: [{ name: 'Анна', id: '1', added: false }] });
    window.prompt = vi.fn().mockReturnValue('');
    result.current.searchFriends();
    expect(update).not.toHaveBeenCalled();
  });

  it('searchFriends toggles friend by name', () => {
    setup({ friends: [{ name: 'Анна', id: '1', added: false }] });
    window.prompt = vi.fn().mockReturnValue('АН');
    result.current.searchFriends();
    expect(update).toHaveBeenCalledWith({
      friends: [{ name: 'Анна', id: '1', added: true }],
      profile: { ...state.profile, friends: 1 },
    });
  });

  it('searchFriends toggles friend by id', () => {
    setup({ friends: [{ name: 'Анна', id: '7', added: true }] });
    window.prompt = vi.fn().mockReturnValue('7');
    result.current.searchFriends();
    expect(update).toHaveBeenCalledWith({
      friends: [{ name: 'Анна', id: '7', added: false }],
      profile: { ...state.profile, friends: 0 },
    });
  });

  it('showDevices adds a device', () => {
    setup();
    window.prompt = vi.fn().mockReturnValue(' Apple Watch ');
    result.current.showDevices();
    expect(update).toHaveBeenCalledWith({
      profile: { ...state.profile, devices: ['Apple Watch'] },
    });
  });

  it('showDevices removes last device', () => {
    setup({ profile: { ...makeState().profile, devices: ['Watch', 'Ring'] } });
    window.prompt = vi.fn().mockReturnValue('');
    result.current.showDevices();
    expect(update).toHaveBeenCalledWith({
      profile: { ...state.profile, devices: ['Watch'] },
    });
  });

  it('showDevices keeps list when empty and blank', () => {
    setup({ profile: { ...makeState().profile, devices: [] } });
    window.prompt = vi.fn().mockReturnValue('');
    result.current.showDevices();
    expect(update).toHaveBeenCalledWith({
      profile: { ...state.profile, devices: [] },
    });
  });

  it('showDevices stops when cancelled', () => {
    setup();
    window.prompt = vi.fn().mockReturnValue(null);
    result.current.showDevices();
    expect(update).not.toHaveBeenCalled();
  });

  it('loadProfile merges backend data', async () => {
    setup();
    api.loadProfile.mockResolvedValue({
      profile: {
        full_name: 'Михаил Б',
        email: 'new@example.com',
        gender: 'male',
        phone: '999',
        bio: 'Бегун',
        status: 'Активен',
      },
    });
    await act(async () => {
      await result.current.loadProfile();
    });
    expect(update).toHaveBeenCalledWith({
      profile: {
        name: 'Михаил Б',
        email: 'new@example.com',
        gender: 'male',
        status: 'Активен',
        bio: 'Бегун',
        phone: '999',
        devices: [],
      },
      profileLoaded: true,
    });
  });

  it('loadProfile falls back to existing fields', async () => {
    setup();
    api.loadProfile.mockResolvedValue({ profile: { full_name: 'Другое' } });
    await act(async () => {
      await result.current.loadProfile();
    });
    expect(update).toHaveBeenCalledWith({
      profile: expect.objectContaining({
        name: 'Другое',
        email: 'mih@example.com',
      }),
      profileLoaded: true,
    });
  });

  it('loadProfile marks loaded when payload empty', async () => {
    setup();
    api.loadProfile.mockResolvedValue({});
    await act(async () => {
      await result.current.loadProfile();
    });
    expect(update).toHaveBeenCalledWith({ profileLoaded: true });
  });

  it('loadProfile marks loaded on error', async () => {
    setup();
    api.loadProfile.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.loadProfile();
    });
    expect(update).toHaveBeenCalledWith({ profileLoaded: true });
  });

  it('saveProfile notifies on success', async () => {
    setup();
    api.saveProfile.mockResolvedValue({});
    let ok;
    await act(async () => {
      ok = await result.current.saveProfile({ full_name: 'Михаил' });
    });
    expect(api.saveProfile).toHaveBeenCalledWith({ full_name: 'Михаил' });
    expect(notify).toHaveBeenCalledWith('Профиль сохранён');
    expect(ok).toBe(true);
  });

  it('saveProfile notifies on failure', async () => {
    setup();
    api.saveProfile.mockRejectedValue(new Error('boom'));
    let ok;
    await act(async () => {
      ok = await result.current.saveProfile({});
    });
    expect(notify).toHaveBeenCalledWith('Ошибка сохранения профиля');
    expect(ok).toBe(false);
  });

  it('uploadPhoto ignores missing file', () => {
    setup();
    result.current.uploadPhoto({ target: { files: [] } });
    expect(update).not.toHaveBeenCalled();
  });

  it('uploadPhoto stores data url', async () => {
    setup();
    const file = new File(['binary'], 'photo.png', { type: 'image/png' });
    await act(async () => {
      result.current.uploadPhoto({ target: { files: [file] } });
    });
    await waitFor(() => expect(update).toHaveBeenCalled());
    expect(update).toHaveBeenCalledWith({
      profileImage: expect.stringContaining('data:'),
    });
  });

  it('toggleFriendAdded toggles by name', () => {
    setup({
      friends: [
        { name: 'Анна', id: '1', added: false },
        { name: 'Иван', id: '2', added: true },
      ],
    });
    result.current.toggleFriendAdded('Анна');
    expect(update).toHaveBeenCalledWith({
      friends: [
        { name: 'Анна', id: '1', added: true },
        { name: 'Иван', id: '2', added: true },
      ],
      profile: { ...state.profile, friends: 2 },
    });
  });

  it('toggleFriendAdded toggles by index', () => {
    setup({
      friends: [
        { name: 'Анна', id: '1', added: false },
        { name: 'Иван', id: '2', added: true },
      ],
    });
    result.current.toggleFriendAdded(1);
    expect(update).toHaveBeenCalledWith({
      friends: [
        { name: 'Анна', id: '1', added: false },
        { name: 'Иван', id: '2', added: false },
      ],
      profile: { ...state.profile, friends: 0 },
    });
  });

  it('toggleFriendAdded ignores unknown friend', () => {
    setup({ friends: [] });
    result.current.toggleFriendAdded('Анна');
    expect(update).not.toHaveBeenCalled();
  });

  it('openFriendProfile selects friend by name', () => {
    const friend = { name: 'Анна', id: '1' };
    setup({ friends: [friend] });
    result.current.openFriendProfile('Анна');
    expect(update).toHaveBeenCalledWith({ selectedFriend: friend });
  });

  it('openFriendProfile selects friend by object', () => {
    const friend = { name: 'Анна', id: '1' };
    setup({ friends: [] });
    result.current.openFriendProfile(friend);
    expect(update).toHaveBeenCalledWith({ selectedFriend: friend });
  });

  it('openFriendProfile notifies when missing', () => {
    setup({ friends: [] });
    result.current.openFriendProfile('Нет');
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Пользователь не найден');
  });

  it('showAchievement selects achievement', () => {
    const ach = { name: 'Водный баланс' };
    setup({ achievements: [ach] });
    result.current.showAchievement('Водный баланс');
    expect(update).toHaveBeenCalledWith({ selectedAchievement: ach });
  });

  it('showAchievement ignores unknown achievement', () => {
    setup({ achievements: [] });
    result.current.showAchievement('Нет');
    expect(update).not.toHaveBeenCalled();
  });

  it('showAchievementsList clears selection', () => {
    setup();
    result.current.showAchievementsList();
    expect(update).toHaveBeenCalledWith({ selectedAchievement: null });
  });
});
