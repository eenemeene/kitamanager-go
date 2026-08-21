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
    const amendFn = jest.fn();

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract, unknown>({
          createFn,
          amendFn,
          toAmendData: ({ from, ...rest }) => ({ effective_from: from, ...rest }),
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
    expect(amendFn).not.toHaveBeenCalled();
    // ...and it must not claim to have ended a contract it never touched. The
    // toast used to read off the checkbox rather than off what happened, so a
    // plain create announced "previous contract ended" on a screen whose whole
    // subject is contract history.
    expect(mockToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'contracts.createSuccess' })
    );
  });

  it('reports an amendment when one actually happened', async () => {
    const createFn = jest.fn();
    const amendFn = jest.fn().mockResolvedValue({ closed: {}, successor: {} });

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract, unknown>({
          createFn,
          amendFn,
          toAmendData: ({ from, ...rest }) => ({ effective_from: from, ...rest }),
          invalidateQueryKeys: [['children', 1]],
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        entityId: 5,
        data: { from: '2026-01-01', section_id: 1 },
        entity: {
          contracts: [{ id: 50, version: 3, from: '2020-01-01', to: null }],
        },
        endCurrentContract: true,
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(amendFn).toHaveBeenCalled();
    expect(mockToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'contracts.previousContractEnded' })
    );
  });

  it('amends the active contract, passing its version and the chosen effective date', async () => {
    const createFn = jest.fn();
    const amendFn = jest.fn().mockResolvedValue({ id: 11, from: '2026-01-01', section_id: 2 });

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract, unknown>({
          createFn,
          amendFn,
          toAmendData: ({ from, ...rest }) => ({ effective_from: from, ...rest }),
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
          contracts: [{ id: 99, version: 3, from: today, to: futureDate }],
        },
        endCurrentContract: true,
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    // The version travels as the If-Match precondition, and the form's `from`
    // becomes effective_from instead of being stripped — the old behaviour meant
    // the server started the successor today, ignoring the date the user picked.
    // 2026-06-01 is the date the form supplied. It reaches the server as
    // effective_from, where the old code stripped it and the server silently
    // started the successor today — so the user picked one date and got another.
    expect(amendFn).toHaveBeenCalledWith(5, 99, 3, {
      effective_from: '2026-06-01',
      section_id: 2,
    });
    expect(createFn).not.toHaveBeenCalled();
  });

  it('invalidates all query keys including extraInvalidateKeys on success', async () => {
    const createFn = jest.fn().mockResolvedValue({ id: 10, from: '2026-01-01', section_id: 1 });
    const primaryKey = ['children', 1];
    const unpaginatedKey = ['childrenAll', 1];
    const statsKey = ['contractProperties', 1];

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract, unknown>({
          createFn,
          amendFn: jest.fn(),
          toAmendData: ({ from, ...rest }) => ({ effective_from: from, ...rest }),
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
        useContractMutation<TestCreateData, TestUpdateData, TestContract, unknown>({
          createFn,
          amendFn: jest.fn(),
          toAmendData: ({ from, ...rest }) => ({ effective_from: from, ...rest }),
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

  it('falls back to createFn when endCurrentContract=true but no active contract exists', async () => {
    // Edge case worth its own test: the user toggles "end current contract"
    // on but the entity has no currently-active contract (e.g. all contracts
    // are in the future or already ended). The hook must still create a new
    // one rather than no-op or crash trying to update nothing.
    const createFn = jest.fn().mockResolvedValue({ id: 30, from: '2026-01-01', section_id: 1 });
    const amendFn = jest.fn();

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract, unknown>({
          createFn,
          amendFn,
          toAmendData: ({ from, ...rest }) => ({ effective_from: from, ...rest }),
          invalidateQueryKeys: [['children', 1]],
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({
        entityId: 5,
        data: { from: '2026-01-01', section_id: 1 },
        // Entity has only a past contract — no active one for "today".
        entity: {
          contracts: [{ id: 50, version: 3, from: '2020-01-01', to: '2020-06-30' }],
        },
        endCurrentContract: true,
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(createFn).toHaveBeenCalledWith(5, { from: '2026-01-01', section_id: 1 });
    expect(amendFn).not.toHaveBeenCalled();
  });

  it('calls onSuccess callback', async () => {
    const createFn = jest.fn().mockResolvedValue({ id: 10, from: '2026-01-01', section_id: 1 });
    const onSuccess = jest.fn();

    const { result } = renderHook(
      () =>
        useContractMutation<TestCreateData, TestUpdateData, TestContract, unknown>({
          createFn,
          amendFn: jest.fn(),
          toAmendData: ({ from, ...rest }) => ({ effective_from: from, ...rest }),
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
