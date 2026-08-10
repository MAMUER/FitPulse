import { apiRequest } from './client';

export async function addBiometricRecord(
  metricType,
  value,
  timestamp,
  deviceType
) {
  return apiRequest('/biometrics', {
    method: 'POST',
    body: JSON.stringify({
      metric_type: metricType,
      value,
      timestamp,
      device_type: deviceType,
    }),
  });
}

export async function getBiometricRecords(metricType, from, to, limit = 100) {
  let url = `/biometrics?metric_type=${metricType}&limit=${limit}`;
  if (from) url += `&from=${from}`;
  if (to) url += `&to=${to}`;
  return apiRequest(url);
}

export async function listHealthConditions(conditionType = '') {
  let url = '/health/conditions';
  if (conditionType)
    url += `?condition_type=${encodeURIComponent(conditionType)}`;
  return apiRequest(url);
}

export async function upsertHealthCondition(data) {
  return apiRequest('/health/conditions', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteHealthCondition(conditionId) {
  return apiRequest(`/health/conditions/${conditionId}`, { method: 'DELETE' });
}

export async function listBodyComposition(from, to, limit = 100) {
  let url = `/health/body-composition?limit=${limit}`;
  if (from) url += `&from=${from}`;
  if (to) url += `&to=${to}`;
  return apiRequest(url);
}

export async function createBodyComposition(data) {
  return apiRequest('/health/body-composition', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function listMenstrualCycles() {
  return apiRequest('/health/menstrual-cycles');
}

export async function createMenstrualCycle(data) {
  return apiRequest('/health/menstrual-cycles', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateMenstrualCycle(cycleId, data) {
  return apiRequest(`/health/menstrual-cycles/${cycleId}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteMenstrualCycle(cycleId) {
  return apiRequest(`/health/menstrual-cycles/${cycleId}`, { method: 'DELETE' });
}
