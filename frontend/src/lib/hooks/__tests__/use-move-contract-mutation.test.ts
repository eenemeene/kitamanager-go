import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { useMoveContractMutation } from '../use-move-contract-mutation';
import { createTestQueryClient, createHookWrapper } from '@/test-utils';

const mockToast = jest.fn();
// Both entry points: the hook reports success through useToast(), and
// showErrorToast reaches for the module-level toast() so it can be called from
// outside a component.
jest.mock('../use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
  toast: (...args: unknown[]) => mockToast(...args),
}));

interface TestEntity {
  id: number;
  name: string;
  contracts?: { id: number; section_id: number; version: number }[];
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
      { id: 10, name: 'Alice', contracts: [{ id: 100, section_id: 1, version: 1 }] },
      { id: 11, name: 'Bob', contracts: [{ id: 101, section_id: 1, version: 1 }] },
    ];
    queryClient.setQueryData(allKey, entities);

    const amendFn = jest.fn().mockResolvedValue({});

    const { result } = renderHook(
      () =>
        useMoveContractMutation<TestEntity>({
          orgId: 1,
          amendFn,
          correctFn: jest.fn(),
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
      result.current.mutate({
        entityId: 10,
        contractId: 100,
        sectionId: 2,
        version: 1,
        from: '2020-01-01',
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    // The trailing 1 is the contract version, sent as the If-Match precondition.
    // The section move amends by default, so the effective date travels too:
    // which section someone was in is history the occupancy reports read.
    expect(amendFn).toHaveBeenCalledWith(10, 100, 2, 1, expect.any(String));
  });

  it('rolls back optimistic update on error', async () => {
    const allKey = ['childrenAll', 1];
    const entities: TestEntity[] = [
      { id: 10, name: 'Alice', contracts: [{ id: 100, section_id: 1, version: 1 }] },
    ];
    queryClient.setQueryData(allKey, entities);

    const amendFn = jest.fn().mockRejectedValue(new Error('Network error'));

    const { result } = renderHook(
      () =>
        useMoveContractMutation<TestEntity>({
          orgId: 1,
          amendFn,
          correctFn: jest.fn(),
          allUnpaginatedKey: allKey,
          invalidateKeys: () => [['children', 1]],
          successMessage: 'sections.movedSuccess',
          errorMessage: 'sections.movedFailed',
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        entityId: 10,
        contractId: 100,
        sectionId: 5,
        version: 1,
        from: '2020-01-01',
      });
    });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    // Should have rolled back
    const cached = queryClient.getQueryData<TestEntity[]>(allKey);
    expect(cached?.[0].contracts?.[0].section_id).toBe(1);

    // A network failure carries no message of its own, so the configured one is
    // what the user gets.
    expect(mockToast).toHaveBeenCalledWith(
      expect.objectContaining({ description: 'sections.movedFailed', variant: 'destructive' })
    );
  });

  it('tells the user why the server refused, not just that it did', async () => {
    // Dragging a card corrects or amends a contract, and the refusals carry
    // information worth showing. The stale-version one is the case a board left
    // open will actually hit, and "Failed to move child" does not tell anyone
    // what to do about it.
    const allKey = ['childrenAll', 1];
    queryClient.setQueryData(allKey, [
      { id: 10, name: 'Alice', contracts: [{ id: 100, section_id: 1, version: 1 }] },
    ] satisfies TestEntity[]);

    const staleVersion = {
      response: {
        data: {
          type: 'u',
          title: 'Resource was modified',
          status: 412,
          code: 'precondition_failed',
          detail:
            'this contract was changed by someone else (you have version 1, current is 2) — reload and reapply your change',
        },
      },
    };

    const { result } = renderHook(
      () =>
        useMoveContractMutation<TestEntity>({
          orgId: 1,
          amendFn: jest.fn().mockRejectedValue(staleVersion),
          correctFn: jest.fn().mockRejectedValue(staleVersion),
          allUnpaginatedKey: allKey,
          invalidateKeys: () => [['children', 1]],
          successMessage: 'sections.movedSuccess',
          errorMessage: 'sections.movedFailed',
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        entityId: 10,
        contractId: 100,
        sectionId: 5,
        version: 1,
        from: '2020-01-01',
      });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(mockToast).toHaveBeenCalledWith(
      expect.objectContaining({
        description: expect.stringContaining('changed by someone else'),
        variant: 'destructive',
      })
    );
    // And the optimistic move is still undone.
    const cached = queryClient.getQueryData<TestEntity[]>(allKey);
    expect(cached?.[0].contracts?.[0].section_id).toBe(1);
  });

  it('invalidates all keys including entity-specific keys on settled', async () => {
    const allKey = ['childrenAll', 1];
    queryClient.setQueryData(allKey, [
      { id: 10, name: 'Alice', contracts: [{ id: 100, section_id: 1, version: 1 }] },
    ]);

    const amendFn = jest.fn().mockResolvedValue({});

    const { result } = renderHook(
      () =>
        useMoveContractMutation<TestEntity>({
          orgId: 1,
          amendFn,
          correctFn: jest.fn(),
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
      result.current.mutate({
        entityId: 10,
        contractId: 100,
        sectionId: 3,
        version: 1,
        from: '2020-01-01',
      });
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
