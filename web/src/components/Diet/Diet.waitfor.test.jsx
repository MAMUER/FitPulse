import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

describe('Diet check', () => {
  it('waitFor works', async () => {
    render(<div>Hello</div>);
    await waitFor(() =>
      expect(screen.getByText('Hello')).toBeDefined()
    );
  });
});
