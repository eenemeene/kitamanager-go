import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { useContractMutation } from '../use-contract-mutation';
import { createTestQueryClient, createHookWrapper } from '@/test-utils';

const mockToast = jest.fn();
jest.mock('../use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

const mockShowErrorToast = jest.fn();
jest.mock('@/lib/utils/show-error-toast', () => ({
  showErrorToast: (...args: unknown[]) => mockShowErrorToast(...args),
}));

interface TestContract {
  id: number;
  from: string;
  to?: string | null;
  section_id: number;
}

interface TestCreateData {
  from: string;
  section_id: number;
}

interface TestUpdateData {
  section_id: number;
}

describe('useContractMutation', () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createHookWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createHookWrapper(queryClient);
    mockToast.mockClear();
    mockShowErrorToast.mockClear();
    jest.spyOn(queryClient, 'invalidateQueries');
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('calls createFn when endCurrentContract is false', async () => {
    const createFn = jest.fn().mockResolvedValue({ id: 10, from: '2026-01-01', section_id: 1 });
    const updateFn = jest.fn();

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract>({
          createFn,
          updateFn,
          toUpdateData: ({ from, ...rest }) => rest,
          invalidateQueryKeys: [['children', 1]],
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        entityId: 5,
        data: { from: '2026-01-01', section_id: 1 },
        entity: null,
        endCurrentContract: false,
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(createFn).toHaveBeenCalledWith(5, { from: '2026-01-01', section_id: 1 });
    expect(updateFn).not.toHaveBeenCalled();
  });

  it('calls updateFn when endCurrentContract is true and active contract exists', async () => {
    const createFn = jest.fn();
    const updateFn = jest.fn().mockResolvedValue({ id: 11, from: '2026-01-01', section_id: 2 });

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract>({
          createFn,
          updateFn,
          toUpdateData: ({ from, ...rest }) => rest,
          invalidateQueryKeys: [['children', 1]],
        }),
      { wrapper }
    );

    const today = new Date().toISOString().split('T')[0];
    const futureDate = '2099-12-31';

    act(() => {
      result.current.mutate({
        entityId: 5,
        data: { from: '2026-06-01', section_id: 2 },
        entity: {
          contracts: [{ id: 99, from: today, to: futureDate }],
        },
        endCurrentContract: true,
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(updateFn).toHaveBeenCalledWith(5, 99, { section_id: 2 });
    expect(createFn).not.toHaveBeenCalled();
  });

  it('invalidates all query keys including extraInvalidateKeys on success', async () => {
    const createFn = jest.fn().mockResolvedValue({ id: 10, from: '2026-01-01', section_id: 1 });
    const primaryKey = ['children', 1];
    const unpaginatedKey = ['childrenAll', 1];
    const statsKey = ['contractProperties', 1];

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract>({
          createFn,
          updateFn: jest.fn(),
          toUpdateData: ({ from, ...rest }) => rest,
          invalidateQueryKeys: [primaryKey, unpaginatedKey, statsKey],
          extraInvalidateKeys: (entityId) => [
            ['childContracts', 1, entityId],
            ['child', 1, entityId],
          ],
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        entityId: 7,
        data: { from: '2026-01-01', section_id: 1 },
        entity: null,
        endCurrentContract: false,
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    // Primary keys
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['children', 1] });
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['childrenAll', 1] });
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['contractProperties', 1],
    });
    // Extra keys with entityId
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['childContracts', 1, 7],
    });
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['child', 1, 7] });
  });

  it('shows error toast on failure', async () => {
    const createFn = jest.fn().mockRejectedValue(new Error('Server error'));

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract>({
          createFn,
          updateFn: jest.fn(),
          toUpdateData: ({ from, ...rest }) => rest,
          invalidateQueryKeys: [['children', 1]],
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        entityId: 5,
        data: { from: '2026-01-01', section_id: 1 },
        entity: null,
        endCurrentContract: false,
      });
    });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    expect(mockShowErrorToast).toHaveBeenCalledWith(
      'common.error',
      expect.any(Error),
      'common.failedToCreate'
    );
  });

  it('calls onSuccess callback', async () => {
    const createFn = jest.fn().mockResolvedValue({ id: 10, from: '2026-01-01', section_id: 1 });
    const onSuccess = jest.fn();

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract>({
          createFn,
          updateFn: jest.fn(),
          toUpdateData: ({ from, ...rest }) => rest,
          invalidateQueryKeys: [['children', 1]],
          onSuccess,
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        entityId: 5,
        data: { from: '2026-01-01', section_id: 1 },
        entity: null,
        endCurrentContract: false,
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(onSuccess).toHaveBeenCalled();
  });
});
