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
  MouseSensor: jest.fn(),
  TouchSensor: jest.fn(),
  KeyboardSensor: jest.fn(),
  useSensor: jest.fn(() => ({})),
  useSensors: jest.fn(() => []),
  useDroppable: () => ({
    setNodeRef: jest.fn(),
    isOver: false,
  }),
  useDraggable: ({ disabled }: { disabled?: boolean } = {}) => ({
    // Mirrors what dnd-kit does: a disabled draggable still gets these back,
    // which is why the cards withhold them rather than relying on `disabled`
    // alone to stop announcing themselves as movable.
    attributes: { role: 'button', tabIndex: 0, 'aria-roledescription': 'draggable' },
    listeners: { onPointerDown: jest.fn() },
    setNodeRef: jest.fn(),
    isDragging: false,
    disabled: !!disabled,
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
    // server (the backend's active_on filter narrows the result
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
  // -------------------------------------------------------------------------
  // Snapshot date
  // -------------------------------------------------------------------------
  //
  // The board fetches for its as-of date, but it used to bucket the cards it
  // got back by whichever contract was active *today*. So the one thing the
  // date picker exists for did not work: a child who had since changed rooms
  // appeared under the room they are in now, and one who had since left had no
  // contract active today at all and was dropped from the board without a
  // trace. The fetch was covered; the bucketing was not.
  describe('as-of date', () => {
    const sectionsFixture = {
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
          min_age_months: 36,
          max_age_months: 72,
          created_at: '2024-01-01T00:00:00Z',
          created_by: 'admin',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ],
      total: 2,
      page: 1,
      limit: 100,
      total_pages: 1,
    };

    // Moved from Krippe to Mäuse on 2025-08-01, and left on 2026-07-31.
    const movedAndLeft = [
      {
        id: 1,
        organization_id: 1,
        first_name: 'Emma',
        last_name: 'Schmidt',
        gender: 'female' as const,
        birthdate: '2022-06-15',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        vouchers: [],
        contracts: [
          {
            id: 1,
            child_id: 1,
            version: 1,
            from: '2024-08-01T00:00:00Z',
            to: '2025-07-31T00:00:00Z',
            section_id: 1,
            section_name: 'Krippe',
            properties: {},
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
          {
            id: 2,
            child_id: 1,
            version: 1,
            from: '2025-08-01T00:00:00Z',
            to: '2026-07-31T00:00:00Z',
            section_id: 2,
            section_name: 'Mäuse',
            properties: {},
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
      },
    ];

    /** Which column heading precedes a card in document order. */
    function columnOf(name: string): string | null {
      const card = screen.getByText(name);
      const column = card.closest('div.w-72');
      return column?.querySelector('h3')?.textContent ?? null;
    }

    beforeEach(() => {
      mockApiClient.getSections.mockResolvedValue(sectionsFixture);
      mockApiClient.getChildrenAllForDate.mockResolvedValue(movedAndLeft);
      mockApiClient.getEmployeesAllForDate.mockResolvedValue([]);
    });

    /** Render the board and move its date picker to `date`. */
    async function boardAsOf(date: string) {
      const { fireEvent } = await import('@testing-library/react');
      render(<SectionKanbanBoard orgId={1} />, { wrapper: TestWrapper });
      // Not on the board at all to begin with: both Emma's contracts have
      // ended, so nothing covers today. That is the point -- the card exists
      // only on a date she was enrolled for.
      await screen.findByText('sections.dragHint');
      expect(screen.queryByText('Emma Schmidt')).not.toBeInTheDocument();

      const dateInput = screen.getByLabelText('sections.asOfDate') as HTMLInputElement;
      fireEvent.change(dateInput, { target: { value: date } });
      await screen.findByDisplayValue(date);
    }

    it('buckets a card by the contract in force on the chosen date', async () => {
      await boardAsOf('2025-01-15');
      expect(columnOf('Emma Schmidt')).toBe('Krippe');
    });

    it('follows the same child to the room a later contract moved them to', async () => {
      await boardAsOf('2026-01-15');
      expect(columnOf('Emma Schmidt')).toBe('Mäuse');
    });

    it('keeps a child who has since left on the board, in the room they were in', async () => {
      await boardAsOf('2025-01-15');
      // No contract covers today, so resolving against today answered null and
      // the card was silently dropped off the board.
      expect(screen.getByText('Emma Schmidt')).toBeInTheDocument();
      expect(columnOf('Emma Schmidt')).toBe('Krippe');
    });

    it('is a read-only snapshot away from today', async () => {
      await boardAsOf('2025-01-15');

      // The hint swaps, and the cards stop offering themselves as draggable:
      // the only write the board can make is "amend from today", which does not
      // describe a card being looked at on another date.
      expect(screen.getByText('sections.snapshotHint')).toBeInTheDocument();
      expect(screen.queryByText('sections.dragHint')).not.toBeInTheDocument();
      expect(screen.getByText('Emma Schmidt').closest('[role="button"]')).toBeNull();
    });

    it('offers the cards again once the date is back to today', async () => {
      const { fireEvent } = await import('@testing-library/react');
      await boardAsOf('2025-01-15');
      expect(screen.getByText('sections.snapshotHint')).toBeInTheDocument();

      fireEvent.click(screen.getByText('sections.backToToday'));

      expect(await screen.findByText('sections.dragHint')).toBeInTheDocument();
    });
  });
});
