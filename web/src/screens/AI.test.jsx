import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Ai from './AI';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('AI', () => {
  it('sends message and shows results', () => {
    const sendAI = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: {
        messages: [
          {
            id: 1,
            type: 'user',
            text: 'hi',
            classification: { a: 1 },
            plan: { b: 2 },
            diet: { c: 3 },
          },
          { id: 2, type: 'bot', text: 'ok' },
        ],
        aiClassification: { x: 1 },
        aiPlan: { y: 2 },
        aiDiet: { z: 3 },
      },
      t: (ru, _en) => ru,
      sendAI,
      go,
    });

    render(<Ai />);
    // The composer has no <label>; it is identified by id/placeholder.
    const input = screen.getByPlaceholderText('Например: у меня мало времени');
    fireEvent.change(input, { target: { value: 'test' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(sendAI).toHaveBeenCalled();
    fireEvent.change(input, { target: { value: '' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }));
    expect(screen.getByText('hi')).toBeInTheDocument();
    expect(screen.getByText('ok')).toBeInTheDocument();
  });

  it('renders ai summary with missing plan and diet', async () => {
    const sendAI = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: {
        messages: [],
        aiClassification: { x: 1 },
        aiPlan: null,
        aiDiet: null,
      },
      t: (ru, _en) => ru,
      sendAI,
      go,
    });
    render(<Ai />);
    await screen.findByText('Классификация');
    expect(screen.getByText('Классификация')).toBeInTheDocument();
  });

  it('renders with empty messages', async () => {
    const sendAI = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: { messages: [] },
      t: (ru, _en) => ru,
      sendAI,
      go,
    });
    render(<Ai />);
    await new Promise((r) => setTimeout(r, 50));
    expect(screen.getByText('FitPulse AI')).toBeInTheDocument();
  });

  it('renders without messages', () => {
    const sendAI = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: { messages: undefined, aiClassification: null },
      t: (ru, _en) => ru,
      sendAI,
      go,
    });
    render(<Ai />);
    expect(screen.getByText('FitPulse AI')).toBeInTheDocument();
  });

  it('sends a quick question as a prompt', () => {
    const quickAI = vi.fn();
    useApp.mockReturnValue({
      state: { messages: [] },
      t: (ru, _en) => ru,
      sendAI: vi.fn(),
      quickAI,
      go: vi.fn(),
    });

    render(<Ai />);
    fireEvent.click(screen.getByRole('button', { name: 'Меню' }));
    expect(quickAI).toHaveBeenCalledWith('Составь меню на сегодня');
  });

  it('shows ai summary when classification present without messages', () => {
    const sendAI = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: {
        messages: [],
        aiClassification: { x: 1 },
        aiPlan: { y: 2 },
        aiDiet: { z: 3 },
      },
      t: (ru, _en) => ru,
      sendAI,
      go,
    });
    render(<Ai />);
    expect(screen.getByText('Классификация')).toBeInTheDocument();
  });
});
