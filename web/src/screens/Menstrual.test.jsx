import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Menstrual from './Menstrual';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const cycles = [
  {
    id: 1,
    start_date: '2026-01-05',
    end_date: '2026-01-09',
    flow_intensity: 'moderate',
    symptoms: ['головная боль'],
    moods: ['раздражительность'],
    notes: 'Без замечаний',
  },
  {
    id: 2,
    start_date: '2025-12-01',
    end_date: null,
    flow_intensity: 'light',
  },
];

function createContext(overrides = {}) {
  return {
    state: {
      screen: 'menstrual',
      theme: 'dark',
      highContrast: false,
      language: 'ru',
      menstrualCycles: cycles,
    },
    t: (ru, _en) => ru,
    notify: vi.fn(),
    loadMenstrualCycles: vi.fn().mockResolvedValue({ cycles }),
    createMenstrualCycle: vi.fn().mockResolvedValue({}),
    updateMenstrualCycle: vi.fn().mockResolvedValue({}),
    deleteMenstrualCycle: vi.fn().mockResolvedValue({}),
    ...overrides,
  };
}

function renderMenstrual(context) {
  useApp.mockReturnValue(context);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Menstrual />
      </AppProvider>
    </BrowserRouter>
  );
}

async function fillForm(user, values) {
  fireEvent.change(document.getElementById('start_date'), {
    target: { value: values.start_date },
  });
  fireEvent.change(document.getElementById('flow_intensity'), {
    target: { value: values.flow_intensity || 'moderate' },
  });
  fireEvent.change(document.getElementById('symptoms'), {
    target: { value: values.symptoms || '' },
  });
  fireEvent.change(document.getElementById('moods'), {
    target: { value: values.moods || '' },
  });
  fireEvent.change(document.getElementById('notes'), {
    target: { value: values.notes || '' },
  });
  await user.click(screen.getByRole('button', { name: 'Добавить' }));
}

describe('Menstrual', () => {
  it('renders loading state before cycles arrive', async () => {
    const context = createContext();
    renderMenstrual(context);
    expect(screen.getByText('Загрузка...')).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument()
    );
    expect(context.loadMenstrualCycles).toHaveBeenCalled();
  });

  it('renders panel, disclaimer and cycle list', async () => {
    const context = createContext();
    renderMenstrual(context);
    expect(screen.getByText('Менструальный календарь')).toBeInTheDocument();
    expect(screen.getByText('Ручной ввод и прогнозы')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Это не медицинский совет. Данные носят ознакомительный характер.'
      )
    ).toBeInTheDocument();
    expect(
      await screen.findByText('2026-01-05 → 2026-01-09')
    ).toBeInTheDocument();
    expect(
      screen.getByText('Интенсивность: moderate · Без замечаний')
    ).toBeInTheDocument();
    expect(screen.getByText('2025-12-01')).toBeInTheDocument();
    expect(screen.getByText('Интенсивность: light')).toBeInTheDocument();
    expect(screen.getAllByText('Изменить')).toHaveLength(2);
    expect(screen.getAllByText('Удалить')).toHaveLength(2);
  });

  it('shows next period estimate block for two cycles', async () => {
    renderMenstrual(createContext());
    expect(
      await screen.findByText('Следующий период примерно')
    ).toBeInTheDocument();
    expect(document.querySelector('.metric-value')).toBeInTheDocument();
  });

  it('renders empty state when no cycles exist', async () => {
    const context = createContext({
      state: {
        screen: 'menstrual',
        theme: 'dark',
        highContrast: false,
        language: 'ru',
        menstrualCycles: [],
      },
      loadMenstrualCycles: vi.fn().mockResolvedValue({ cycles: [] }),
    });
    renderMenstrual(context);
    expect(await screen.findByText('Нет записей')).toBeInTheDocument();
  });

  it('notifies when start date is missing', async () => {
    const user = userEvent.setup();
    const context = createContext();
    renderMenstrual(context);
    await waitFor(() =>
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument()
    );
    await user.click(screen.getByRole('button', { name: 'Добавить' }));
    expect(context.notify).toHaveBeenCalledWith('Укажите дату начала цикла');
    expect(context.createMenstrualCycle).not.toHaveBeenCalled();
  });

  it('creates a cycle from the form', async () => {
    const user = userEvent.setup();
    const context = createContext({
      state: {
        screen: 'menstrual',
        theme: 'dark',
        highContrast: false,
        language: 'ru',
        menstrualCycles: [],
      },
      createMenstrualCycle: vi.fn().mockResolvedValue({ cycle: { id: 3 } }),
    });
    renderMenstrual(context);
    await waitFor(() =>
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument()
    );
    await fillForm(user, {
      start_date: '2026-02-10',
      flow_intensity: 'heavy',
      symptoms: 'головная боль, усталость',
      moods: 'тревога',
      notes: 'Плотные дни',
    });
    expect(context.createMenstrualCycle).toHaveBeenCalledWith({
      start_date: '2026-02-10',
      end_date: undefined,
      flow_intensity: 'heavy',
      symptoms: ['головная боль', 'усталость'],
      moods: ['тревога'],
      notes: 'Плотные дни',
    });
    expect(context.notify).toHaveBeenCalledWith('Цикл добавлен');
    expect(document.getElementById('start_date').value).toBe('');
    expect(document.getElementById('notes').value).toBe('');
    expect(screen.getByRole('button', { name: 'Добавить' })).toBeEnabled();
  });

  it('omits optional fields from payload', async () => {
    const user = userEvent.setup();
    const context = createContext({
      state: {
        screen: 'menstrual',
        theme: 'dark',
        highContrast: false,
        language: 'ru',
        menstrualCycles: [],
      },
      createMenstrualCycle: vi.fn().mockResolvedValue({}),
    });
    renderMenstrual(context);
    await waitFor(() =>
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument()
    );
    await fillForm(user, { start_date: '2026-03-01' });
    expect(context.createMenstrualCycle).toHaveBeenCalledWith({
      start_date: '2026-03-01',
      end_date: undefined,
      flow_intensity: 'moderate',
      symptoms: [],
      moods: [],
      notes: undefined,
    });
  });

  it('notifies when saving fails', async () => {
    const user = userEvent.setup();
    const context = createContext({
      state: {
        screen: 'menstrual',
        theme: 'dark',
        highContrast: false,
        language: 'ru',
        menstrualCycles: [],
      },
      createMenstrualCycle: vi.fn().mockRejectedValue(new Error('boom')),
    });
    renderMenstrual(context);
    await waitFor(() =>
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument()
    );
    await fillForm(user, { start_date: '2026-03-01' });
    await waitFor(() =>
      expect(context.notify).toHaveBeenCalledWith('Ошибка сохранения цикла')
    );
  });

  it('edits existing cycle and saves updates', async () => {
    const user = userEvent.setup();
    const context = createContext();
    renderMenstrual(context);
    expect(
      await screen.findByText('2026-01-05 → 2026-01-09')
    ).toBeInTheDocument();
    await user.click(screen.getAllByText('Изменить')[0]);
    expect(document.getElementById('start_date').value).toBe('2026-01-05');
    expect(document.getElementById('end_date').value).toBe('2026-01-09');
    expect(document.getElementById('flow_intensity').value).toBe('moderate');
    expect(document.getElementById('symptoms').value).toBe('головная боль');
    expect(document.getElementById('moods').value).toBe('раздражительность');
    expect(document.getElementById('notes').value).toBe('Без замечаний');
    expect(
      screen.getByRole('button', { name: 'Сохранить' })
    ).toBeInTheDocument();

    fireEvent.change(document.getElementById('flow_intensity'), {
      target: { value: 'light' },
    });
    await user.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() =>
      expect(context.updateMenstrualCycle).toHaveBeenCalledWith(1, {
        start_date: '2026-01-05',
        end_date: '2026-01-09',
        flow_intensity: 'light',
        symptoms: ['головная боль'],
        moods: ['раздражительность'],
        notes: 'Без замечаний',
      })
    );
    expect(context.notify).toHaveBeenCalledWith('Цикл обновлён');
    expect(
      screen.getByRole('button', { name: 'Добавить' })
    ).toBeInTheDocument();
  });

  it('cancels editing without saving', async () => {
    const user = userEvent.setup();
    const context = createContext();
    renderMenstrual(context);
    expect(
      await screen.findByText('2026-01-05 → 2026-01-09')
    ).toBeInTheDocument();
    await user.click(screen.getAllByText('Изменить')[0]);
    await user.click(screen.getByRole('button', { name: 'Отмена' }));
    expect(context.updateMenstrualCycle).not.toHaveBeenCalled();
    expect(document.getElementById('start_date').value).toBe('');
    expect(
      screen.getByRole('button', { name: 'Добавить' })
    ).toBeInTheDocument();
  });

  it('deletes a cycle', async () => {
    const user = userEvent.setup();
    const context = createContext();
    renderMenstrual(context);
    expect(
      await screen.findByText('2026-01-05 → 2026-01-09')
    ).toBeInTheDocument();
    await user.click(screen.getAllByText('Удалить')[1]);
    await waitFor(() =>
      expect(context.deleteMenstrualCycle).toHaveBeenCalledWith(2)
    );
    expect(context.notify).toHaveBeenCalledWith('Цикл удалён');
  });

  it('notifies when deleting fails', async () => {
    const user = userEvent.setup();
    const context = createContext({
      deleteMenstrualCycle: vi.fn().mockRejectedValue(new Error('boom')),
    });
    renderMenstrual(context);
    await waitFor(() =>
      expect(screen.getAllByText('Удалить').length).toBeGreaterThan(0)
    );
    await user.click(screen.getAllByText('Удалить')[0]);
    await waitFor(() =>
      expect(context.notify).toHaveBeenCalledWith('Ошибка удаления цикла')
    );
  });

  it('notifies when loading fails', async () => {
    const context = createContext({
      state: {
        screen: 'menstrual',
        theme: 'dark',
        highContrast: false,
        language: 'ru',
        menstrualCycles: [],
      },
      loadMenstrualCycles: vi.fn().mockRejectedValue(new Error('boom')),
    });
    renderMenstrual(context);
    await waitFor(() =>
      expect(context.notify).toHaveBeenCalledWith('Ошибка загрузки цикла')
    );
  });
});
