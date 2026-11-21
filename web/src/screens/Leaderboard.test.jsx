import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Leaderboard from './Leaderboard';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

function setup(state = {}, overrides = {}) {
  const syncLeaderboard = vi.fn();
  const loadLeaderboard = vi.fn();
  const reportUser = vi.fn();
  useApp.mockReturnValue({
    state,
    t: (ru) => ru,
    syncLeaderboard,
    loadLeaderboard,
    reportUser,
    ...overrides,
  });
  return { syncLeaderboard, loadLeaderboard, reportUser };
}

describe('Leaderboard', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('syncs own points and loads the leaderboard on mount', () => {
    const { syncLeaderboard, loadLeaderboard } = setup({ leaderboard: [] });

    render(<Leaderboard />);

    expect(syncLeaderboard).toHaveBeenCalled();
    expect(loadLeaderboard).toHaveBeenCalled();
  });

  it('shows the empty state when there are no participants', () => {
    setup({ leaderboard: [] });

    render(<Leaderboard />);

    expect(screen.getByText('Пока нет участников')).toBeInTheDocument();
  });

  it('renders entries and reports another user', () => {
    const { reportUser } = setup({
      leaderboard: [
        { user_id: 'u1', display_name: 'Анна', points: 120, me: true },
        { user_id: 'u2', display_name: 'Борис', points: 90, me: false },
      ],
    });

    render(<Leaderboard />);

    expect(screen.getByText('Анна')).toBeInTheDocument();
    expect(screen.getByText('Борис')).toBeInTheDocument();

    // The "me" row has no report button; only the other user does.
    fireEvent.click(screen.getByLabelText('Пожаловаться'));
    expect(reportUser).toHaveBeenCalledWith('u2');
  });

  it('falls back to a default name for entries without display_name', () => {
    setup({ leaderboard: [{ user_id: 'u3', points: 5 }] });

    render(<Leaderboard />);

    expect(screen.getByText('Участник')).toBeInTheDocument();
  });
});
