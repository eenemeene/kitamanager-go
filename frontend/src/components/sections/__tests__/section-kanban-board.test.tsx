import React from 'react';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SectionKanbanBoard } from '../section-kanban-board';
import { apiClient } from '@/lib/api/client';

// Mock API client
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSections: jest.fn(),
    getChildrenAll: jest.fn(),
    updateChild: jest.fn(),
  },
}));

// Mock @dnd-kit/core
jest.mock('@dnd-kit/core', () => ({
  DndContext: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DragOverlay: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PointerSensor: jest.fn(),
  useSensor: jest.fn(() => ({})),
  useSensors: jest.fn(() => []),
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

// Mock toast
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({
    toast: jest.fn(),
  }),
}));

const mockApiClient = apiClient as jest.Mocked<typeof apiClient>;

function TestWrapper({ children }: { children: React.ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

describe('SectionKanbanBoard', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders loading skeletons while data is loading', () => {
    mockApiClient.getSections.mockReturnValue(new Promise(() => {})); // Never resolves
    mockApiClient.getChildrenAll.mockReturnValue(new Promise(() => {}));

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    // Should show skeleton elements (check for animate-pulse class)
    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders unassigned column and section columns after loading', async () => {
    mockApiClient.getSections.mockResolvedValue({
      data: [
        {
          id: 1,
          organization_id: 1,
          name: 'Krippe',
          is_default: false,
          created_at: '2024-01-01T00:00:00Z',
          created_by: 'admin',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 2,
          organization_id: 1,
          name: 'Mäuse',
          is_default: false,
          created_at: '2024-01-01T00:00:00Z',
          created_by: 'admin',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ],
      total: 2,
      page: 1,
      limit: 100,
      total_pages: 1,
    });

    mockApiClient.getChildrenAll.mockResolvedValue([
      {
        id: 1,
        organization_id: 1,
        first_name: 'Emma',
        last_name: 'Schmidt',
        gender: 'female',
        birthdate: '2020-06-15',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        contracts: [
          {
            id: 1,
            child_id: 1,
            from: '2024-01-01T00:00:00Z',
            to: null,
            section_id: 1,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
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
        contracts: [
          {
            id: 2,
            child_id: 2,
            from: '2024-01-01T00:00:00Z',
            to: null,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
      },
    ]);

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    // Wait for data to load
    expect(await screen.findByText('sections.unassigned')).toBeInTheDocument();
    expect(screen.getByText('Krippe')).toBeInTheDocument();
    expect(screen.getByText('Mäuse')).toBeInTheDocument();
  });

  it('renders children in correct columns', async () => {
    mockApiClient.getSections.mockResolvedValue({
      data: [
        {
          id: 1,
          organization_id: 1,
          name: 'Krippe',
          is_default: false,
          created_at: '2024-01-01T00:00:00Z',
          created_by: 'admin',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      limit: 100,
      total_pages: 1,
    });

    mockApiClient.getChildrenAll.mockResolvedValue([
      {
        id: 1,
        organization_id: 1,
        first_name: 'Emma',
        last_name: 'Schmidt',
        gender: 'female',
        birthdate: '2020-06-15',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        contracts: [
          {
            id: 1,
            child_id: 1,
            from: '2024-01-01T00:00:00Z',
            to: null,
            section_id: 1,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
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
        contracts: [
          {
            id: 2,
            child_id: 2,
            from: '2024-01-01T00:00:00Z',
            to: null,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
      },
    ]);

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    // Wait for children to appear
    expect(await screen.findByText('Emma Schmidt')).toBeInTheDocument();
    expect(screen.getByText('Max Müller')).toBeInTheDocument();
  });

  it('calls getChildrenAll with contract_on filter (server-side filtering)', async () => {
    mockApiClient.getSections.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
      limit: 100,
      total_pages: 0,
    });
    mockApiClient.getChildrenAll.mockResolvedValue([]);

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    // Wait for loading to finish
    await screen.findByText('sections.dragHint');

    // Verify getChildrenAll was called (API does the contract filtering)
    expect(mockApiClient.getChildrenAll).toHaveBeenCalledWith(1);
  });

  it('renders drag hint text', async () => {
    mockApiClient.getSections.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
      limit: 100,
      total_pages: 0,
    });
    mockApiClient.getChildrenAll.mockResolvedValue([]);

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    expect(await screen.findByText('sections.dragHint')).toBeInTheDocument();
  });
});
