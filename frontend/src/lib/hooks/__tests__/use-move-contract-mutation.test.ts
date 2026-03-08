import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { useMoveContractMutation } from '../use-move-contract-mutation';
import { createTestQueryClient, createHookWrapper } from '@/test-utils';

const mockToast = jest.fn();
jest.mock('../use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

interface TestEntity {
  id: number;
  name: string;
  contracts?: { id: number; section_id: number }[];
}

describe('useMoveContractMutation', () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createHookWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createHookWrapper(queryClient);
    mockToast.mockClear();
    jest.spyOn(queryClient, 'invalidateQueries');
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('optimistically updates section_id in cache', async () => {
    const allKey = ['childrenAll', 1];
    const entities: TestEntity[] = [
      { id: 10, name: 'Alice', contracts: [{ id: 100, section_id: 1 }] },
      { id: 11, name: 'Bob', contracts: [{ id: 101, section_id: 1 }] },
    ];
    queryClient.setQueryData(allKey, entities);

    const updateFn = jest.fn().mockResolvedValue({});

    const { result } = renderHook(
      () =>
        useMoveContractMutation<TestEntity>({
          orgId: 1,
          updateFn,
          allUnpaginatedKey: allKey,
          invalidateKeys: (entityId) => [
            ['children', 1],
            ['child', 1, entityId],
          ],
          successMessage: 'sections.movedSuccess',
          errorMessage: 'sections.movedFailed',
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ entityId: 10, contractId: 100, sectionId: 2 });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(updateFn).toHaveBeenCalledWith(10, 100, 2);
  });

  it('rolls back optimistic update on error', async () => {
    const allKey = ['childrenAll', 1];
    const entities: TestEntity[] = [
      { id: 10, name: 'Alice', contracts: [{ id: 100, section_id: 1 }] },
    ];
    queryClient.setQueryData(allKey, entities);

    const updateFn = jest.fn().mockRejectedValue(new Error('Network error'));

    const { result } = renderHook(
      () =>
        useMoveContractMutation<TestEntity>({
          orgId: 1,
          updateFn,
          allUnpaginatedKey: allKey,
          invalidateKeys: () => [['children', 1]],
          successMessage: 'sections.movedSuccess',
          errorMessage: 'sections.movedFailed',
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ entityId: 10, contractId: 100, sectionId: 5 });
    });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    // Should have rolled back
    const cached = queryClient.getQueryData<TestEntity[]>(allKey);
    expect(cached?.[0].contracts?.[0].section_id).toBe(1);

    expect(mockToast).toHaveBeenCalledWith({
      title: 'sections.movedFailed',
      variant: 'destructive',
    });
  });

  it('invalidates all keys including entity-specific keys on settled', async () => {
    const allKey = ['childrenAll', 1];
    queryClient.setQueryData(allKey, [
      { id: 10, name: 'Alice', contracts: [{ id: 100, section_id: 1 }] },
    ]);

    const updateFn = jest.fn().mockResolvedValue({});

    const { result } = renderHook(
      () =>
        useMoveContractMutation<TestEntity>({
          orgId: 1,
          updateFn,
          allUnpaginatedKey: allKey,
          invalidateKeys: (entityId) => [
            ['children', 1],
            ['childContracts', 1, entityId],
            ['child', 1, entityId],
          ],
          successMessage: 'sections.movedSuccess',
          errorMessage: 'sections.movedFailed',
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ entityId: 10, contractId: 100, sectionId: 3 });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    // onSettled invalidates allUnpaginatedKey + all invalidateKeys
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: allKey });
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['children', 1] });
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['childContracts', 1, 10],
    });
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['child', 1, 10] });
  });
});
