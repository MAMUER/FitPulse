import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Health from './Health';

const mockListHealthConditions = vi.fn();
const mockListBodyComposition = vi.fn();
const mockListMenstrualCycles = vi.fn();
const mockCreateBodyComposition = vi.fn();
const mockCreateMenstrualCycle = vi.fn();
const mockUpsertHealthCondition = vi.fn();
const mockDeleteHealthCondition = vi.fn();
const mockDeleteMenstrualCycle = vi.fn();

vi.mock('../../utils/api', () => ({
  listHealthConditions: (...args) => mockListHealthConditions(...args),
  listBodyComposition: (...args) => mockListBodyComposition(...args),
  listMenstrualCycles: (...args) => mockListMenstrualCycles(...args),
  createBodyComposition: (...args) => mockCreateBodyComposition(...args),
  createMenstrualCycle: (...args) => mockCreateMenstrualCycle(...args),
  upsertHealthCondition: (...args) => mockUpsertHealthCondition(...args),
  deleteHealthCondition: (...args) => mockDeleteHealthCondition(...args),
  deleteMenstrualCycle: (...args) => mockDeleteMenstrualCycle(...args),
}));

describe('Health', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListHealthConditions.mockResolvedValue([]);
    mockListBodyComposition.mockResolvedValue([]);
    mockListMenstrualCycles.mockResolvedValue([]);
  });

  it('renders loading state initially', () => {
    mockListHealthConditions.mockImplementation(() => new Promise(() => {}));
    render(<Health />);
    expect(screen.getByText(/Загрузка данных здоровья/i)).toBeDefined();
  });

  it('renders health sections after loading', async () => {
    render(<Health />);
    await waitFor(() =>
      expect(screen.getByText('Заболевания и состояния')).toBeDefined()
    );
    expect(screen.getByText('Состав тела')).toBeDefined();
    expect(screen.getByText('Менструальный цикл')).toBeDefined();
  });

  it('shows empty states when no data', async () => {
    render(<Health />);
    await waitFor(() =>
      expect(screen.getByText('Нет добавленных состояний')).toBeDefined()
    );
  });
});
