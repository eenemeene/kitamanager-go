import { screen, waitFor } from '@testing-library/react';
import ChildrenPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders, createMockPaginatedResponse } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getChildren: jest.fn(),
    getChildrenFunding: jest.fn(),
    createChild: jest.fn(),
    updateChild: jest.fn(),
    deleteChild: jest.fn(),
    createChildContract: jest.fn(),
    updateChildContract: jest.fn(),
  },
  getErrorMessage: jest.fn((e: unknown, f: string) => f),
}));

jest.mock('@/lib/hooks/use-funding-attributes', () => ({
  useFundingAttributes: () => ({
    fundingAttributes: [],
    attributesByKey: {},
    isLoading: false,
    hasNoFunding: false,
  }),
}));

jest.mock('@/components/ui/tag-input', () => ({
  PropertyTagInput: () => <div data-testid="tag-input">Tag Input</div>,
}));

const mockChildren = [
  {
    id: 1,
    first_name: 'Max',
    last_name: 'Mustermann',
    gender: 'male',
    birthdate: '2020-03-15T00:00:00Z',
    organization_id: 1,
    contracts: [
      {
        id: 1,
        child_id: 1,
        from: '2023-01-01',
        properties: { care_type: 'ganztag' },
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
    ],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    first_name: 'Emma',
    last_name: 'Schmidt',
    gender: 'female',
    birthdate: '2021-07-20T00:00:00Z',
    organization_id: 1,
    contracts: [],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const mockFundingResponse = { date: '2024-01-01', children: [] };

const mockPaginatedResponse = createMockPaginatedResponse(mockChildren);
const mockEmptyResponse = createMockPaginatedResponse([]);

describe('ChildrenPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (apiClient.getChildrenFunding as jest.Mock).mockResolvedValue(mockFundingResponse);
  });

  it('renders page title', async () => {
    (apiClient.getChildren as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<ChildrenPage />);

    const titles = screen.getAllByText('children.title');
    expect(titles.length).toBeGreaterThanOrEqual(1);
  });

  it('renders new child button', async () => {
    (apiClient.getChildren as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<ChildrenPage />);

    expect(screen.getByText('children.newChild')).toBeInTheDocument();
  });

  it('shows loading skeleton while fetching', async () => {
    (apiClient.getChildren as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    renderWithProviders(<ChildrenPage />);

    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('displays children in table', async () => {
    (apiClient.getChildren as jest.Mock).mockResolvedValue(mockPaginatedResponse);

    renderWithProviders(<ChildrenPage />);

    await waitFor(() => {
      expect(screen.getByText('Max Mustermann')).toBeInTheDocument();
    });

    expect(screen.getByText('Emma Schmidt')).toBeInTheDocument();
  });

  it('shows no results when empty', async () => {
    (apiClient.getChildren as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<ChildrenPage />);

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });
  });

  it('renders search input', async () => {
    (apiClient.getChildren as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<ChildrenPage />);

    expect(screen.getByPlaceholderText('common.search')).toBeInTheDocument();
  });

  it('renders table headers', async () => {
    (apiClient.getChildren as jest.Mock).mockResolvedValue(mockEmptyResponse);

    renderWithProviders(<ChildrenPage />);

    await waitFor(() => {
      expect(screen.getByText('common.noResults')).toBeInTheDocument();
    });

    expect(screen.getByText('common.name')).toBeInTheDocument();
    expect(screen.getByText('gender.label')).toBeInTheDocument();
    expect(screen.getByText('children.birthdate')).toBeInTheDocument();
    expect(screen.getByText('children.age')).toBeInTheDocument();
    expect(screen.getByText('sections.title')).toBeInTheDocument();
    expect(screen.getByText('children.currentContract')).toBeInTheDocument();
    expect(screen.getByText('children.properties')).toBeInTheDocument();
    expect(screen.getByText('children.funding')).toBeInTheDocument();
    expect(screen.getByText('children.requirement')).toBeInTheDocument();
    expect(screen.getByText('common.actions')).toBeInTheDocument();
  });
});
