import { act, render, renderHook } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import {
  PauseOverlay,
  useLivePauseId,
  usePauseState,
  useReducedMotion,
} from './useReducedMotion';

describe('useReducedMotion', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns false when matchMedia is unavailable', () => {
    const originalMatchMedia = window.matchMedia;
    delete window.matchMedia;

    const { result } = renderHook(() => useReducedMotion());
    expect(result.current).toBeUndefined();

    window.matchMedia = originalMatchMedia;
  });

  it('returns matchMedia result on initial render', () => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    const { result } = renderHook(() => useReducedMotion());
    expect(result.current).toBe(true);
  });

  it('updates when matchMedia change event fires', async () => {
    const addEventListener = vi.fn();
    const removeEventListener = vi.fn();

    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener,
      removeEventListener,
    });

    const { result } = renderHook(() => useReducedMotion());
    expect(result.current).toBe(false);

    act(() => {
      const handler = addEventListener.mock.calls[0][1];
      handler({ matches: true });
    });

    expect(result.current).toBe(true);
  });

  it('cleans up event listener on unmount', () => {
    const addEventListener = vi.fn();
    const removeEventListener = vi.fn();

    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener,
      removeEventListener,
    });

    const { unmount } = renderHook(() => useReducedMotion());
    unmount();

    expect(removeEventListener).toHaveBeenCalledWith(
      'change',
      expect.any(Function)
    );
  });
});

describe('usePauseState', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns initial paused state and effectivePaused', () => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    const { result } = renderHook(() => usePauseState(false));
    expect(result.current.paused).toBe(false);
    expect(result.current.effectivePaused).toBe(false);
    expect(result.current.reducedMotion).toBe(false);
  });

  it('returns effectivePaused true when reduced motion is preferred', () => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    const { result } = renderHook(() => usePauseState(false));
    expect(result.current.paused).toBe(false);
    expect(result.current.effectivePaused).toBe(true);
    expect(result.current.reducedMotion).toBe(true);
  });

  it('returns effectivePaused true when paused is true', () => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    const { result } = renderHook(() => usePauseState(false));
    act(() => {
      result.current.setPaused(true);
    });
    expect(result.current.paused).toBe(true);
    expect(result.current.effectivePaused).toBe(true);
  });

  it('returns effectivePaused true when both paused and reduced motion are true', () => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    const { result } = renderHook(() => usePauseState(false));
    act(() => {
      result.current.setPaused(true);
    });
    expect(result.current.paused).toBe(true);
    expect(result.current.reducedMotion).toBe(true);
    expect(result.current.effectivePaused).toBe(true);
  });
});

describe('useLivePauseId', () => {
  it('returns a unique id', () => {
    const { result: result1 } = renderHook(() => useLivePauseId());
    const { result: result2 } = renderHook(() => useLivePauseId());

    expect(result1.current).toBeDefined();
    expect(result2.current).toBeDefined();
    expect(result1.current).not.toBe(result2.current);
  });
});

describe('PauseOverlay', () => {
  beforeEach(() => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });
  });

  it('renders resume label when paused is true', () => {
    const onToggle = vi.fn();
    const { getByRole } = render(
      <PauseOverlay onToggle={onToggle} paused={true} />
    );

    expect(getByRole('button')).toHaveTextContent('▶ Обновление остановлено');
    expect(getByRole('button')).toHaveAttribute('aria-pressed', 'true');
  });

  it('renders pause label when paused is false', () => {
    const onToggle = vi.fn();
    const { getByRole } = render(
      <PauseOverlay onToggle={onToggle} paused={false} />
    );

    expect(getByRole('button')).toHaveTextContent('⏸ Остановить обновление');
    expect(getByRole('button')).toHaveAttribute('aria-pressed', 'false');
  });

  it('calls onToggle when clicked', async () => {
    const onToggle = vi.fn();
    const { getByRole } = render(
      <PauseOverlay onToggle={onToggle} paused={false} />
    );

    const user = userEvent.setup();
    await user.click(getByRole('button'));

    expect(onToggle).toHaveBeenCalledTimes(1);
  });
});
