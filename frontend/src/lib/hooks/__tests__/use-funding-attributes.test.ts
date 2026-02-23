import { renderHook, waitFor } from '@testing-library/react';
import { useFundingAttributes } from '../use-funding-attributes';
import { createTestQueryClient, createHookWrapper } from '@/test-utils';
import { QueryClient } from '@tanstack/react-query';

// Mock apiClient
const mockGetGovernmentFundings = jest.fn();
const mockGetGovernmentFunding = jest.fn();

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    getGovernmentFundings: (...args: unknown[]) => mockGetGovernmentFundings(...args),
    getGovernmentFunding: (...args: unknown[]) => mockGetGovernmentFunding(...args),
  },
}));

// Mock useUiStore
const mockUseUiStore = jest.fn();
jest.mock('@/stores/ui-store', () => ({
  useUiStore: (selector: (state: unknown) => unknown) => mockUseUiStore(selector),
}));

describe('useFundingAttributes', () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createHookWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createHookWrapper(queryClient);
    jest.clearAllMocks();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('returns empty attributes when no organization found', () => {
    // No orgs -> org is undefined -> state is undefined
    mockUseUiStore.mockImplementation(
      (selector: (state: { organizations: unknown[] }) => unknown) =>
        selector({ organizations: [] })
    );

    const { result } = renderHook(() => useFundingAttributes(999), { wrapper });

    expect(result.current.fundingAttributes).toEqual([]);
    expect(result.current.attributesByKey).toEqual({});
    expect(result.current.isLoading).toBe(false);
    expect(result.current.hasNoFunding).toBe(false);
  });

  it('returns empty when no matching funding for state', async () => {
    mockUseUiStore.mockImplementation(
      (selector: (state: { organizations: { id: number; state: string }[] }) => unknown) =>
        selector({ organizations: [{ id: 1, state: 'berlin' }] })
    );

    // Return fundings that do NOT match 'berlin'
    mockGetGovernmentFundings.mockResolvedValue({
      data: [{ id: 10, name: 'Bayern Funding', state: 'bayern' }],
      total: 1,
      page: 1,
      limit: 100,
      total_pages: 1,
    });

    const { result } = renderHook(() => useFundingAttributes(1, '2024-01-01', '2024-12-31'), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.hasNoFunding).toBe(true);
    });

    expect(result.current.fundingAttributes).toEqual([]);
    expect(result.current.attributesByKey).toEqual({});
  });

  it('returns attributes from overlapping periods', async () => {
    mockUseUiStore.mockImplementation(
      (selector: (state: { organizations: { id: number; state: string }[] }) => unknown) =>
        selector({ organizations: [{ id: 1, state: 'berlin' }] })
    );

    mockGetGovernmentFundings.mockResolvedValue({
      data: [{ id: 10, name: 'Berlin Funding', state: 'berlin' }],
      total: 1,
      page: 1,
      limit: 100,
      total_pages: 1,
    });

    mockGetGovernmentFunding.mockResolvedValue({
      id: 10,
      name: 'Berlin Funding',
      state: 'berlin',
      periods: [
        {
          id: 1,
          government_funding_id: 10,
          from: '2024-01-01',
          to: '2024-12-31',
          created_at: '',
          properties: [
            {
              id: 1,
              period_id: 1,
              key: 'care_type',
              value: 'Ganztag',
              payment: 100,
              requirement: 0,
              created_at: '',
            },
            {
              id: 2,
              period_id: 1,
              key: 'care_type',
              value: 'Halbtag',
              payment: 50,
              requirement: 0,
              created_at: '',
            },
            {
              id: 3,
              period_id: 1,
              key: 'supplement',
              value: 'NDH',
              payment: 20,
              requirement: 0,
              created_at: '',
            },
          ],
        },
        {
          // Non-overlapping period - should NOT appear
          id: 2,
          government_funding_id: 10,
          from: '2025-01-01',
          to: '2025-12-31',
          created_at: '',
          properties: [
            {
              id: 4,
              period_id: 2,
              key: 'care_type',
              value: 'Teilzeit',
              payment: 30,
              requirement: 0,
              created_at: '',
            },
          ],
        },
      ],
    });

    const { result } = renderHook(() => useFundingAttributes(1, '2024-06-01', '2024-06-30'), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.fundingAttributes.length).toBeGreaterThan(0);
    });

    // Only the overlapping period's properties should appear
    const values = result.current.fundingAttributes.map((a) => a.value);
    expect(values).toContain('ganztag');
    expect(values).toContain('halbtag');
    expect(values).toContain('ndh');
    expect(values).not.toContain('teilzeit');
  });

  it('groups attributes by key', async () => {
    mockUseUiStore.mockImplementation(
      (selector: (state: { organizations: { id: number; state: string }[] }) => unknown) =>
        selector({ organizations: [{ id: 1, state: 'berlin' }] })
    );

    mockGetGovernmentFundings.mockResolvedValue({
      data: [{ id: 10, name: 'Berlin Funding', state: 'berlin' }],
      total: 1,
      page: 1,
      limit: 100,
      total_pages: 1,
    });

    mockGetGovernmentFunding.mockResolvedValue({
      id: 10,
      name: 'Berlin Funding',
      state: 'berlin',
      periods: [
        {
          id: 1,
          government_funding_id: 10,
          from: '2024-01-01',
          to: '2024-12-31',
          created_at: '',
          properties: [
            {
              id: 1,
              period_id: 1,
              key: 'care_type',
              value: 'Ganztag',
              payment: 100,
              requirement: 0,
              created_at: '',
            },
            {
              id: 2,
              period_id: 1,
              key: 'care_type',
              value: 'Halbtag',
              payment: 50,
              requirement: 0,
              created_at: '',
            },
            {
              id: 3,
              period_id: 1,
              key: 'supplement',
              value: 'NDH',
              payment: 20,
              requirement: 0,
              created_at: '',
            },
          ],
        },
      ],
    });

    const { result } = renderHook(() => useFundingAttributes(1, '2024-01-01', '2024-12-31'), {
      wrapper,
    });

    await waitFor(() => {
      expect(Object.keys(result.current.attributesByKey).length).toBeGreaterThan(0);
    });

    expect(result.current.attributesByKey['care_type']).toHaveLength(2);
    expect(result.current.attributesByKey['supplement']).toHaveLength(1);
  });

  it('returns isLoading=true when state exists but details not loaded', () => {
    mockUseUiStore.mockImplementation(
      (selector: (state: { organizations: { id: number; state: string }[] }) => unknown) =>
        selector({ organizations: [{ id: 1, state: 'berlin' }] })
    );

    // Fundings query never resolves -> fundingDetails is undefined
    mockGetGovernmentFundings.mockReturnValue(new Promise(() => {}));

    const { result } = renderHook(() => useFundingAttributes(1), { wrapper });

    // state is 'berlin' (truthy) and fundingDetails is undefined -> isLoading = true
    expect(result.current.isLoading).toBe(true);
  });

  it('returns defaultProperties from properties with apply_to_all_contracts', async () => {
    mockUseUiStore.mockImplementation(
      (selector: (state: { organizations: { id: number; state: string }[] }) => unknown) =>
        selector({ organizations: [{ id: 1, state: 'berlin' }] })
    );

    mockGetGovernmentFundings.mockResolvedValue({
      data: [{ id: 10, name: 'Berlin Funding', state: 'berlin' }],
      total: 1,
      page: 1,
      limit: 100,
      total_pages: 1,
    });

    mockGetGovernmentFunding.mockResolvedValue({
      id: 10,
      name: 'Berlin Funding',
      state: 'berlin',
      periods: [
        {
          id: 1,
          government_funding_id: 10,
          from: '2024-01-01',
          to: '2024-12-31',
          created_at: '',
          properties: [
            {
              id: 1,
              period_id: 1,
              key: 'care_type',
              value: 'Ganztag',
              payment: 100,
              requirement: 0,
              created_at: '',
              apply_to_all_contracts: false,
            },
            {
              id: 2,
              period_id: 1,
              key: 'parent',
              value: 'meals',
              label: 'Meals',
              payment: 23,
              requirement: 0,
              created_at: '',
              apply_to_all_contracts: true,
            },
          ],
        },
      ],
    });

    const { result } = renderHook(() => useFundingAttributes(1, '2024-06-01', '2024-06-30'), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.fundingAttributes.length).toBeGreaterThan(0);
    });

    expect(result.current.defaultProperties).toEqual({ parent: 'meals' });
  });

  it('returns empty defaultProperties when no auto-apply properties exist', async () => {
    mockUseUiStore.mockImplementation(
      (selector: (state: { organizations: { id: number; state: string }[] }) => unknown) =>
        selector({ organizations: [{ id: 1, state: 'berlin' }] })
    );

    mockGetGovernmentFundings.mockResolvedValue({
      data: [{ id: 10, name: 'Berlin Funding', state: 'berlin' }],
      total: 1,
      page: 1,
      limit: 100,
      total_pages: 1,
    });

    mockGetGovernmentFunding.mockResolvedValue({
      id: 10,
      name: 'Berlin Funding',
      state: 'berlin',
      periods: [
        {
          id: 1,
          government_funding_id: 10,
          from: '2024-01-01',
          to: '2024-12-31',
          created_at: '',
          properties: [
            {
              id: 1,
              period_id: 1,
              key: 'care_type',
              value: 'Ganztag',
              payment: 100,
              requirement: 0,
              created_at: '',
            },
          ],
        },
      ],
    });

    const { result } = renderHook(() => useFundingAttributes(1, '2024-06-01', '2024-06-30'), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.fundingAttributes.length).toBeGreaterThan(0);
    });

    expect(result.current.defaultProperties).toEqual({});
  });

  it('returns hasNoFunding=true when state exists but no funding matches', async () => {
    mockUseUiStore.mockImplementation(
      (selector: (state: { organizations: { id: number; state: string }[] }) => unknown) =>
        selector({ organizations: [{ id: 1, state: 'hamburg' }] })
    );

    mockGetGovernmentFundings.mockResolvedValue({
      data: [{ id: 10, name: 'Berlin Funding', state: 'berlin' }],
      total: 1,
      page: 1,
      limit: 100,
      total_pages: 1,
    });

    const { result } = renderHook(() => useFundingAttributes(1), { wrapper });

    await waitFor(() => {
      expect(result.current.hasNoFunding).toBe(true);
    });

    expect(result.current.fundingAttributes).toEqual([]);
  });
});
