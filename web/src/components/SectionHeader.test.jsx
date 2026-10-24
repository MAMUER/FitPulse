import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import SectionHeader from './SectionHeader';

describe('SectionHeader', () => {
  it('renders title', () => {
    render(<SectionHeader title='Test Title' />);
    expect(screen.getByText('Test Title')).toBeInTheDocument();
  });

  it('renders right content', () => {
    render(<SectionHeader title='Title' right={<span>Right</span>} />);
    expect(screen.getByText('Right')).toBeInTheDocument();
  });

  it('renders without title', () => {
    render(<SectionHeader right={<span>Right</span>} />);
    expect(screen.getByText('Right')).toBeInTheDocument();
  });
});