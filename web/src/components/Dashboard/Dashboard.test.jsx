import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Dashboard from './Dashboard';

const mockGetBiometricRecords = vi.fn();
const mockClassifyState = vi.fn();
const mockGetTrainingPlans = vi.fn();
const mockGetPlan = vi.fn();

vi.mock('../../utils/api', () => ({
  getBiometricRecords: (...args) => mockGetBiometricRecords(...args),
  classifyState: (...args) => mockClassifyState(...args),
  getTrainingPlans: (...args) => mockGetTrainingPlans(...args),
  getPlan: (...args) => mockGetPlan(...args),
}));

vi.mock('../../hooks/useReducedMotion.jsx', () => ({
  usePauseState: () => ({ effectivePaused: false }),
}));

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders dashboard with loading state', () => {
    mockGetBiometricRecords.mockResolvedValue({ records: [] });
    mockClassifyState.mockResolvedValue({});
    mockGetTrainingPlans.mockResolvedValue({ plans: [] });
    render(<Dashboard />);
    expect(screen.getByText('Динамика пульса')).toBeDefined();
  });

  it('loads biometric data', async () => {
    mockGetBiometricRecords.mockResolvedValue({ records: [{ value: 72 }] });
    mockClassifyState.mockResolvedValue({});
    mockGetTrainingPlans.mockResolvedValue({ plans: [] });
    render(<Dashboard />);
    await waitFor(() => expect(mockGetBiometricRecords).toHaveBeenCalled());
  });

  it('renders AI section', async () => {
    mockGetBiometricRecords.mockResolvedValue({ records: [{ value: 72 }] });
    mockClassifyState.mockResolvedValue({ predicted_class_ru: 'Норма' });
    mockGetTrainingPlans.mockResolvedValue({ plans: [] });
    render(<Dashboard />);
    await waitFor(() => expect(screen.getByText('Норма')).toBeDefined());
  });

  it('renders health summary cards', async () => {
    mockGetBiometricRecords.mockResolvedValue({ records: [{ value: 72 }] });
    mockClassifyState.mockResolvedValue({});
    mockGetTrainingPlans.mockResolvedValue({ plans: [] });
    render(<Dashboard />);
    expect(screen.getByText('Пульс')).toBeDefined();
    expect(screen.getByText('SpO₂')).toBeDefined();
    expect(screen.getByText('Сон')).toBeDefined();
    expect(screen.getByText('Давление')).toBeDefined();
  });
});
