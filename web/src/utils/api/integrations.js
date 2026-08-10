import { apiRequest } from './client';

export async function getProviders() {
  return apiRequest('/integrations/providers');
}

export async function disconnectIntegration(source) {
  return apiRequest(`/integrations/${source}/disconnect`, {
    method: 'POST',
  });
}
