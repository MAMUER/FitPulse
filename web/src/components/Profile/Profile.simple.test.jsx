import { render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';

const mockGetProfile = vi.fn();

vi.mock('../../utils/api', () => ({
  getProfile: () => mockGetProfile(),
}));

function SimpleComponent() {
  const [text, setText] = React.useState('loading');
  React.useEffect(() => {
    mockGetProfile().then((data) => {
      setText(data.text);
    });
  }, []);
  return <div>{text}</div>;
}

describe('Simple component', () => {
  it('updates text', async () => {
    mockGetProfile.mockResolvedValue({ text: 'hello' });
    render(<SimpleComponent />);
    await waitFor(() =>
      expect(screen.queryByText('loading')).not.toBeInTheDocument()
    );
    expect(screen.getByText('hello')).toBeDefined();
  });
});
