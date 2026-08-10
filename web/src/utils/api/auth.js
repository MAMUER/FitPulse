import { apiRequest, setAuthToken } from './client';

export async function login(email, password) {
  const data = await apiRequest('/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
  if (data.access_token) {
    setAuthToken(data.access_token);
  }
  return data;
}

export async function register(email, password, fullName, role = 'client') {
  return apiRequest('/register', {
    method: 'POST',
    body: JSON.stringify({ email, password, full_name: fullName, role }),
  });
}

export async function registerWithInvite(code, name, email, password) {
  return apiRequest('/register/invite', {
    method: 'POST',
    body: JSON.stringify({
      invite_code: code,
      full_name: name,
      email,
      password,
    }),
  });
}

export async function validateInvite(code) {
  return apiRequest('/invite/validate', {
    method: 'POST',
    body: JSON.stringify({ code }),
  });
}

export async function logout() {
  try {
    await apiRequest('/logout', { method: 'POST' });
  } catch (error) {
    console.warn('Logout request failed, clearing token anyway:', error);
  } finally {
    setAuthToken(null);
  }
}

export async function confirmEmail(token) {
  return apiRequest('/auth/confirm-email', {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
}

export async function get2FAStatus() {
  return apiRequest('/auth/2fa/status');
}

export async function setup2FA() {
  return apiRequest('/auth/2fa/setup', { method: 'POST' });
}

export async function confirm2FA(passcode, tempSecret, backupCodes) {
  return apiRequest('/auth/2fa/confirm', {
    method: 'POST',
    body: JSON.stringify({
      passcode,
      temp_secret: tempSecret,
      backup_codes: backupCodes,
    }),
  });
}

export async function verify2FA(tempToken, passcode, isBackupCode = false) {
  return apiRequest('/auth/2fa/verify', {
    method: 'POST',
    body: JSON.stringify({
      temp_token: tempToken,
      passcode,
      is_backup_code: isBackupCode,
    }),
  });
}

export async function disable2FA(passcode) {
  return apiRequest('/auth/2fa/disable', {
    method: 'POST',
    body: JSON.stringify({ passcode }),
  });
}
