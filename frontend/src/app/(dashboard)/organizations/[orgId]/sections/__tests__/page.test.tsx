import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import SectionsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getSections: jest.fn(),
    createSection: jest.fn(),
    updateSection: jest.fn(),
    deleteSection: jest.fn(),
  },
  getErrorMessage: jest.fn((error, fallback) => fallback),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/components/sections/section-kanban-board', () => ({
  SectionKanbanBoard: () => <div data-testid="kanban-board">Kanban Board</div>,
}));

const mockSections = [
  {
    id: 1,
    name: 'Section A',
    organization_id: 1,
    is_default: true,
    created_at: '2024-01-01T00:00:00Z',
    created_by: 'admin',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'Section B',
    organization_id: 1,
    is_default: false,
    created_at: '2024-01-01T00:00:00Z',
    created_by: 'admin',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const mockPaginatedResponse = createMockPaginatedResponse(mockSections);
const mockEmptyResponse = createMockPaginatedResponse([]);

describe('SectionsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders page title', async () => {
    (apiClient.getSections as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<SectionsPage />);

    const titles = screen.getAllByText('sections.title');
    expect(titles.length).toBeGreaterThanOrEqual(1);
  });

  it('renders tab triggers', async () => {
    (apiClient.getSections as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<SectionsPage />);

    expect(screen.getByText('sections.board')).toBeInTheDocument();
    expect(screen.getByText('sections.manage')).toBeInTheDocument();
  });

  it('displays sections in manage tab', async () => {
    (apiClient.getSections as jest.Mock).mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<SectionsPage />);

    // Click the manage tab to show the table
    const manageTab = screen.getByText('sections.manage');
    await userEvent.click(manageTab);

    await waitFor(() => {
      expect(screen.getByText('Section A')).toBeInTheDocument();
    });

    expect(screen.getByText('Section B')).toBeInTheDocument();
  });

  it('shows default section badge', async () => {
    (apiClient.getSections as jest.Mock).mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<SectionsPage />);

    // Click the manage tab to show the table
    const manageTab = screen.getByText('sections.manage');
    await userEvent.click(manageTab);

    await waitFor(() => {
      expect(screen.getByText('Section A')).toBeInTheDocument();
    });

    expect(screen.getByText('sections.defaultSection')).toBeInTheDocument();
  });

  it('renders kanban board mock', async () => {
    (apiClient.getSections as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<SectionsPage />);

    // The board tab is the default tab, so kanban should be visible
    expect(screen.getByTestId('kanban-board')).toBeInTheDocument();
  });
});
