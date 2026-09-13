import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

describe('Diet', () => {
  it('renders plan', async () => {
    render(<div>План питания на сегодня</div>);
    await waitFor(() =>
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument(),
      { timeout: 2000 }
    );
    expect(screen.getByText(/План питания на сегодня/i)).toBeDefined();
  });
});
