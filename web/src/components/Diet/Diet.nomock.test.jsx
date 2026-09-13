import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import Diet from './Diet';

describe('Diet', () => {
  it('renders loading', async () => {
    render(<Diet />);
    expect(screen.getByText('Загрузка...')).toBeDefined();
  });

  it('renders plan', async () => {
    render(<Diet />);
    await waitFor(() =>
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument()
    );
    expect(screen.getByText(/План питания на сегодня/i)).toBeDefined();
  });
});
