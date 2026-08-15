import { apiRequest } from './client';

export async function listInvites(page = 1, pageSize = 10, used = '') {
  const params = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  });
  if (used !== '') params.set('used', String(used));
  return apiRequest(`/invites?${params.toString()}`);
}

export async function createInvite(role, specialty, maxUses) {
  return apiRequest('/invites', {
    method: 'POST',
    body: JSON.stringify({ role, specialty, max_uses: maxUses }),
  });
}

export async function revokeInvite(code) {
  return apiRequest(`/invites/${code}/revoke`, { method: 'POST' });
}

export async function listUsers(page = 1, pageSize = 10) {
  return apiRequest(`/admin/users?page=${page}&page_size=${pageSize}`);
}
