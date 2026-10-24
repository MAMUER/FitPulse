import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import MetricCard from './MetricCard';

describe('MetricCard', () => {
  it('renders value and label', () => {
    render(<MetricCard value='72' label='Heart rate' />);
    expect(screen.getByText('72')).toBeInTheDocument();
    expect(screen.getByText('Heart rate')).toBeInTheDocument();
  });

  it('renders suffix when provided', () => {
    render(<MetricCard value='120' label='BP' suffix='mmHg' />);
    expect(screen.getByText('mmHg')).toBeInTheDocument();
  });
});
