import { screen, waitFor } from '@testing-library/react';
import ChildContractsPage from '../page';
import { apiClient } from '@/lib/api/client';
import { renderWithProviders } from '@/test-utils';

jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1', childId: '1' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string, params?: Record<string, unknown>) => {
      if (params) return `${key}`;
      return key;
    };
    t.has = () => false;
    return t;
  },
}));

jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({ toast: jest.fn() }),
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getChild: jest.fn(),
    getChildContracts: jest.fn(),
    createChildContract: jest.fn(),
    updateChildContract: jest.fn(),
    deleteChildContract: jest.fn(),
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

const mockChild = {
  id: 1,
  first_name: 'Max',
  last_name: 'Mustermann',
  gender: 'male',
  birthdate: '2020-03-15T00:00:00Z',
  organization_id: 1,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const mockContracts = [
  {
    id: 1,
    child_id: 1,
    from: '2023-01-01T00:00:00Z',
    to: null,
    properties: { care_type: 'ganztag' },
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

describe('ChildContractsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders page title with child name', async () => {
    (apiClient.getChild as jest.Mock).mockResolvedValue(mockChild);
    (apiClient.getChildContracts as jest.Mock).mockResolvedValue(mockContracts);

    renderWithProviders(<ChildContractsPage />);

    await waitFor(() => {
      expect(screen.getByText('Max Mustermann')).toBeInTheDocument();
    });

    const historyLabels = screen.getAllByText('children.contractHistory');
    expect(historyLabels.length).toBeGreaterThanOrEqual(1);
  });

  it('shows loading skeleton', async () => {
    (apiClient.getChild as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );
    (apiClient.getChildContracts as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    renderWithProviders(<ChildContractsPage />);

    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('displays contracts in table', async () => {
    (apiClient.getChild as jest.Mock).mockResolvedValue(mockChild);
    (apiClient.getChildContracts as jest.Mock).mockResolvedValue(mockContracts);

    renderWithProviders(<ChildContractsPage />);

    await waitFor(() => {
      expect(screen.getByText('Max Mustermann')).toBeInTheDocument();
    });

    // Contract properties should be displayed
    expect(screen.getByText('ganztag')).toBeInTheDocument();
  });

  it('shows no contracts message when empty', async () => {
    (apiClient.getChild as jest.Mock).mockResolvedValue(mockChild);
    (apiClient.getChildContracts as jest.Mock).mockResolvedValue([]);

    renderWithProviders(<ChildContractsPage />);

    await waitFor(() => {
      expect(screen.getByText('Max Mustermann')).toBeInTheDocument();
    });

    const noContractsMessages = screen.getAllByText('children.noContractsFound');
    expect(noContractsMessages.length).toBeGreaterThanOrEqual(1);
  });

  it('renders back button', async () => {
    (apiClient.getChild as jest.Mock).mockResolvedValue(mockChild);
    (apiClient.getChildContracts as jest.Mock).mockResolvedValue(mockContracts);

    renderWithProviders(<ChildContractsPage />);

    await waitFor(() => {
      expect(screen.getByText('Max Mustermann')).toBeInTheDocument();
    });

    // Back button is a ghost button with ArrowLeft icon
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThanOrEqual(1);
  });
});
