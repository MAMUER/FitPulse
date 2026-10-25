import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import CookieConsent from './CookieConsent';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('CookieConsent', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    localStorage.clear();
  });

  it('shows when not accepted', async () => {
    useApp.mockReturnValue({ t: (ru, _en) => ru });
    localStorage.removeItem('cookie-consent');
    render(<CookieConsent />);
    await screen.findByText('Принять');
    fireEvent.click(screen.getByText('Принять'));
    expect(localStorage.getItem('cookie-consent')).toBe('accepted');
  });

  it('shows when declined not stored', async () => {
    useApp.mockReturnValue({ t: (_ru, en) => en });
    localStorage.removeItem('cookie-consent');
    render(<CookieConsent />);
    await screen.findByText('Decline');
    fireEvent.click(screen.getByText('Decline'));
    expect(localStorage.getItem('cookie-consent')).toBe('declined');
  });

  it('does not show when accepted', () => {
    useApp.mockReturnValue({ t: (ru, _en) => ru });
    localStorage.setItem('cookie-consent', 'accepted');
    const { container } = render(<CookieConsent />);
    expect(container.firstChild).toBeNull();
  });

  it('returns true when localStorage throws on init', () => {
    useApp.mockReturnValue({ t: (ru, _en) => ru });
    const orig = localStorage.getItem;
    localStorage.getItem = () => {
      throw new Error('boom');
    };
    const { container } = render(<CookieConsent />);
    localStorage.getItem = orig;
    expect(container.firstChild).toBeNull();
  });
});
