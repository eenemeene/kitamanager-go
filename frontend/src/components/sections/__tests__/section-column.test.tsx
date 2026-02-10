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
    section_id: 1,
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
    section_id: 1,
  },
];

describe('SectionColumn', () => {
  it('renders section name', () => {
    render(<SectionColumn id="1" title="Krippe" items={mockChildren} />);
    expect(screen.getByText('Krippe')).toBeInTheDocument();
  });

  it('renders child count badge', () => {
    render(<SectionColumn id="1" title="Krippe" items={mockChildren} />);
    expect(screen.getByText('2')).toBeInTheDocument();
  });

  it('renders children cards', () => {
    render(<SectionColumn id="1" title="Krippe" items={mockChildren} />);
    expect(screen.getByText('Emma Schmidt')).toBeInTheDocument();
    expect(screen.getByText('Max Müller')).toBeInTheDocument();
  });

  it('renders empty state when no children', () => {
    render(<SectionColumn id="1" title="Krippe" items={[]} />);
    expect(screen.getByText('common.noResults')).toBeInTheDocument();
  });

  it('renders zero count badge when empty', () => {
    render(<SectionColumn id="1" title="Krippe" items={[]} />);
    expect(screen.getByText('0')).toBeInTheDocument();
  });

  it('renders default badge when isDefault is true', () => {
    render(<SectionColumn id="1" title="Krippe" items={[]} isDefault={true} />);
    expect(screen.getByText('sections.defaultSection')).toBeInTheDocument();
  });

  it('does not render default badge when isDefault is false', () => {
    render(<SectionColumn id="1" title="Krippe" items={[]} isDefault={false} />);
    expect(screen.queryByText('sections.defaultSection')).not.toBeInTheDocument();
  });
});
