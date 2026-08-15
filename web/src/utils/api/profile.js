import { apiRequest } from './client';

export async function getProfile() {
  return apiRequest('/profile');
}

export async function updateProfile(profile) {
  return apiRequest('/profile', {
    method: 'PUT',
    body: JSON.stringify(profile),
  });
}

export async function changePassword(currentPassword, newPassword) {
  return apiRequest('/auth/change-password', {
    method: 'POST',
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });
}

export async function changeEmail(newEmail, password) {
  return apiRequest('/auth/change-email', {
    method: 'POST',
    body: JSON.stringify({ new_email: newEmail, password }),
  });
}

export async function deleteProfile(password) {
  return apiRequest('/profile', {
    method: 'DELETE',
    body: JSON.stringify({ password }),
  });
}
