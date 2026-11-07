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
    fireEvent.change(screen.getByLabelText('Запрос'), {
      target: { value: 'test' },
    });
    fireEvent.keyDown(screen.getByLabelText('Запрос'), { key: 'Enter' });
    expect(sendAI).toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('Запрос'), {
      target: { value: '' },
    });
    fireEvent.click(screen.getByText('Отправить'));
    fireEvent.click(screen.getByText('Назад'));
    expect(go).toHaveBeenCalledWith('home');
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
    expect(screen.getByText('AI-советник')).toBeInTheDocument();
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
    expect(screen.getByText('AI-советник')).toBeInTheDocument();
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
