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
  contracts: [],
  vouchers: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

// The global next-intl mock returns the key, so asserting on keys is how these
// tests check that a string goes through the catalogue at all — which is the
// point here: the card used to render "years" and "M"/"F"/"D" as English
// literals, in an app whose primary audience is German.
describe('ChildCard', () => {
  it('renders child full name', () => {
    render(<ChildCard child={mockChild} />);
    expect(screen.getByText('Emma Schmidt')).toBeInTheDocument();
  });

  it.each([
    ['female', 'gender.short.female', 'gender.female'],
    ['male', 'gender.short.male', 'gender.male'],
    ['diverse', 'gender.short.diverse', 'gender.diverse'],
  ] as const)('translates the %s gender badge', (gender, shortKey, longKey) => {
    render(<ChildCard child={{ ...mockChild, gender }} />);
    expect(screen.getByText(shortKey)).toBeInTheDocument();
    // The abbreviation is for width; a screen reader gets the whole word.
    expect(screen.getByText(longKey)).toHaveClass('sr-only');
  });

  it('renders the age through the catalogue rather than as English', () => {
    render(<ChildCard child={mockChild} />);
    expect(screen.getByText('sections.childAge')).toBeInTheDocument();
    expect(screen.queryByText(/\byears?\b/)).not.toBeInTheDocument();
  });
});
