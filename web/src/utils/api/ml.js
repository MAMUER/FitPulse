import { apiRequest } from './client';

export async function classifyState(biometrics) {
  return apiRequest('/ml/classify', {
    method: 'POST',
    body: JSON.stringify(biometrics),
  });
}

export async function generateMLPlan(
  trainingClass,
  userProfile,
  goal,
  constraints
) {
  return apiRequest('/ml/generate-plan', {
    method: 'POST',
    body: JSON.stringify({
      training_class: trainingClass,
      user_profile: userProfile,
      goal,
      constraints,
    }),
  });
}
