import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

describe('Simple check', () => {
  it('renders simple div', async () => {
    render(<div>Hello</div>);
    expect(screen.getByText('Hello')).toBeDefined();
    await waitFor(() =>
      expect(screen.queryByText('Bye')).not.toBeInTheDocument()
    );
  });
});
