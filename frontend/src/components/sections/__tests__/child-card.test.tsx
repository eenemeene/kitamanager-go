import React from 'react';
import { render, screen } from '@testing-library/react';
import { ChildCard } from '../child-card';
import type { Child } from '@/lib/api/types';

// Mock @dnd-kit/core
jest.mock('@dnd-kit/core', () => ({
  useDraggable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: jest.fn(),
    isDragging: false,
  }),
}));

const mockChild: Child = {
  id: 1,
  organization_id: 1,
  first_name: 'Emma',
  last_name: 'Schmidt',
  gender: 'female',
  birthdate: '2020-06-15',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  section_id: null,
};

describe('ChildCard', () => {
  it('renders child full name', () => {
    render(<ChildCard child={mockChild} />);
    expect(screen.getByText('Emma Schmidt')).toBeInTheDocument();
  });

  it('renders gender badge for female', () => {
    render(<ChildCard child={mockChild} />);
    expect(screen.getByText('F')).toBeInTheDocument();
  });

  it('renders gender badge for male', () => {
    const maleChild: Child = { ...mockChild, gender: 'male' };
    render(<ChildCard child={maleChild} />);
    expect(screen.getByText('M')).toBeInTheDocument();
  });

  it('renders gender badge for diverse', () => {
    const diverseChild: Child = { ...mockChild, gender: 'diverse' };
    render(<ChildCard child={diverseChild} />);
    expect(screen.getByText('D')).toBeInTheDocument();
  });

  it('renders age from birthdate', () => {
    // Child born in 2020, so age should be calculated
    render(<ChildCard child={mockChild} />);
    // The age text should contain "years"
    const ageText = screen.getByText(/\d+ years?/);
    expect(ageText).toBeInTheDocument();
  });
});
