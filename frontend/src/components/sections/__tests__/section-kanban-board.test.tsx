import React from 'react';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SectionKanbanBoard } from '../section-kanban-board';
import { apiClient } from '@/lib/api/client';

// Mock API client
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSections: jest.fn(),
    // The board now uses date-aware fetchers so the user can shift
    // the snapshot date — `For Date` variants take an explicit
    // YYYY-MM-DD instead of defaulting to today inside the client.
    getChildrenAllForDate: jest.fn().mockResolvedValue([]),
    getEmployeesAllForDate: jest.fn().mockResolvedValue([]),
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
    mockApiClient.getChildrenAllForDate.mockReturnValue(new Promise(() => {}));

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    // Should show skeleton elements (check for animate-pulse class)
    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders section columns after loading', async () => {
    mockApiClient.getSections.mockResolvedValue({
      data: [
        {
          id: 1,
          organization_id: 1,
          name: 'Krippe',
          is_default: false,
          min_age_months: 0,
          max_age_months: 36,
          created_at: '2024-01-01T00:00:00Z',
          created_by: 'admin',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 2,
          organization_id: 1,
          name: 'Mäuse',
          is_default: false,
          min_age_months: 0,
          max_age_months: 36,
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

    mockApiClient.getChildrenAllForDate.mockResolvedValue([
      {
        id: 1,
        organization_id: 1,
        first_name: 'Emma',
        last_name: 'Schmidt',
        gender: 'female',
        birthdate: '2020-06-15',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        vouchers: [],
        contracts: [
          {
            id: 1,
            child_id: 1,
            version: 1,
            from: '2024-01-01T00:00:00Z',
            to: '',
            section_id: 1,
            section_name: 'Krippe',
            properties: {},
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
        vouchers: [],
        contracts: [
          {
            id: 2,
            child_id: 2,
            version: 1,
            from: '2024-01-01T00:00:00Z',
            to: '',
            section_id: 2,
            section_name: 'Mäuse',
            properties: {},
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
      },
    ]);

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    // Wait for data to load
    expect(await screen.findByText('Krippe')).toBeInTheDocument();
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
          min_age_months: 0,
          max_age_months: 36,
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

    mockApiClient.getChildrenAllForDate.mockResolvedValue([
      {
        id: 1,
        organization_id: 1,
        first_name: 'Emma',
        last_name: 'Schmidt',
        gender: 'female',
        birthdate: '2020-06-15',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        vouchers: [],
        contracts: [
          {
            id: 1,
            child_id: 1,
            version: 1,
            from: '2024-01-01T00:00:00Z',
            to: '',
            section_id: 1,
            section_name: 'Krippe',
            properties: {},
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
        vouchers: [],
        contracts: [
          {
            id: 2,
            child_id: 2,
            version: 1,
            from: '2024-01-01T00:00:00Z',
            to: '',
            section_id: 1,
            section_name: 'Krippe',
            properties: {},
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

  it('calls getChildrenAllForDate with the asOf date (default: today)', async () => {
    mockApiClient.getSections.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
      limit: 100,
      total_pages: 0,
    });
    mockApiClient.getChildrenAllForDate.mockResolvedValue([]);

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    // Wait for loading to finish
    await screen.findByText('sections.dragHint');

    // The board defaults to today and passes the date through to the
    // server (the backend's contract_on filter narrows the result
    // set, so the page doesn't have to load every contract ever).
    // Locking in (orgId, YYYY-MM-DD) shape so a future "let's just
    // pass orgId" PR is exposed.
    expect(mockApiClient.getChildrenAllForDate).toHaveBeenCalledWith(1, expect.any(String));
    const calledWith = mockApiClient.getChildrenAllForDate.mock.calls[0][1];
    expect(calledWith).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it('refetches when the asOf date changes', async () => {
    // Lock-in for the S8 date-shift feature: changing the date in
    // the picker triggers a refetch through the date-aware
    // fetchers. Without the date in the query key, TanStack Query
    // would serve the stale today-snapshot and the user would see
    // unchanged data.
    const { fireEvent } = await import('@testing-library/react');
    mockApiClient.getSections.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
      limit: 100,
      total_pages: 0,
    });
    mockApiClient.getChildrenAllForDate.mockResolvedValue([]);
    mockApiClient.getEmployeesAllForDate.mockResolvedValue([]);

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });
    await screen.findByText('sections.dragHint');

    const initialChildCalls = mockApiClient.getChildrenAllForDate.mock.calls.length;

    // Change the date — the picker is the only date input on the
    // page so getByDisplayValue is fine.
    const dateInput = screen.getByLabelText('sections.asOfDate') as HTMLInputElement;
    fireEvent.change(dateInput, { target: { value: '2024-01-15' } });

    // Wait for the refetch.
    await screen.findByDisplayValue('2024-01-15');

    // The fetcher must have been called again, with the new date.
    expect(mockApiClient.getChildrenAllForDate.mock.calls.length).toBeGreaterThan(
      initialChildCalls
    );
    const lastCall =
      mockApiClient.getChildrenAllForDate.mock.calls[
        mockApiClient.getChildrenAllForDate.mock.calls.length - 1
      ];
    expect(lastCall).toEqual([1, '2024-01-15']);
  });

  it('renders drag hint text', async () => {
    mockApiClient.getSections.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
      limit: 100,
      total_pages: 0,
    });
    mockApiClient.getChildrenAllForDate.mockResolvedValue([]);

    render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });

    expect(await screen.findByText('sections.dragHint')).toBeInTheDocument();
  });
});
