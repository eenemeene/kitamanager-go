import React from 'react';
import { render, screen } from '@testing-library/react';
import { SectionColumn } from '../section-column';
import type { Child } from '@/lib/api/types';

// Mock @dnd-kit/core
jest.mock('@dnd-kit/core', () => ({
  useDroppable: () => ({
    setNodeRef: jest.fn(),
    isOver: false,
  }),
  useDraggable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: jest.fn(),
    isDragging: false,
  }),
}));

const mockChildren: Child[] = [
  {
    id: 1,
    organization_id: 1,
    first_name: 'Emma',
    last_name: 'Schmidt',
    gender: 'female',
    birthdate: '2020-06-15',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    organization_id: 1,
    first_name: 'Max',
    last_name: 'Müller',
    gender: 'male',
    birthdate: '2021-03-20',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

describe('SectionColumn', () => {
  it('renders section name', () => {
    render(<SectionColumn id="1" title="Krippe" items={mockChildren} employees={[]} />);
    expect(screen.getByText('Krippe')).toBeInTheDocument();
  });

  it('renders separate employee and child count badges', () => {
    render(<SectionColumn id="1" title="Krippe" items={mockChildren} employees={[]} />);
    expect(screen.getByText('0 E')).toBeInTheDocument();
    expect(screen.getByText('2 K')).toBeInTheDocument();
  });

  it('renders children cards', () => {
    render(<SectionColumn id="1" title="Krippe" items={mockChildren} employees={[]} />);
    expect(screen.getByText('Emma Schmidt')).toBeInTheDocument();
    expect(screen.getByText('Max Müller')).toBeInTheDocument();
  });

  it('renders empty state when no children', () => {
    render(<SectionColumn id="1" title="Krippe" items={[]} employees={[]} />);
    expect(screen.getByText('common.noResults')).toBeInTheDocument();
  });

  it('renders zero count badges when empty', () => {
    render(<SectionColumn id="1" title="Krippe" items={[]} employees={[]} />);
    expect(screen.getByText('0 E')).toBeInTheDocument();
    expect(screen.getByText('0 K')).toBeInTheDocument();
  });

  it('renders default badge when isDefault is true', () => {
    render(<SectionColumn id="1" title="Krippe" items={[]} employees={[]} isDefault={true} />);
    expect(screen.getByText('sections.defaultSection')).toBeInTheDocument();
  });

  it('does not render default badge when isDefault is false', () => {
    render(<SectionColumn id="1" title="Krippe" items={[]} employees={[]} isDefault={false} />);
    expect(screen.queryByText('sections.defaultSection')).not.toBeInTheDocument();
  });
});
