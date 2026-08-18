import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { useCrudMutations } from '../use-crud-mutations';
import { createTestQueryClient, createHookWrapper } from '@/test-utils';

// Mock the toast hook
const mockToast = jest.fn();
jest.mock('../use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

// Mock showErrorToast
const mockShowErrorToast = jest.fn();
jest.mock('@/lib/utils/show-error-toast', () => ({
  showErrorToast: (...args: unknown[]) => mockShowErrorToast(...args),
}));

interface TestItem {
  id: number;
  name: string;
}

interface TestCreateData {
  name: string;
}

interface TestUpdateData {
  name?: string;
}

describe('useCrudMutations', () => {
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

  describe('createMutation', () => {
    it('calls createFn and shows success toast on success', async () => {
      const mockCreateFn = jest.fn().mockResolvedValue({ id: 1, name: 'New Item' });
      const mockOnSuccess = jest.fn();
      const mockOnCreateSuccess = jest.fn();

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            createFn: mockCreateFn,
            onSuccess: mockOnSuccess,
            onCreateSuccess: mockOnCreateSuccess,
          }),
        { wrapper }
      );

      act(() => {
        result.current.createMutation.mutate({ name: 'New Item' });
      });

      await waitFor(() => {
        expect(result.current.createMutation.isSuccess).toBe(true);
      });

      expect(mockCreateFn).toHaveBeenCalledWith({ name: 'New Item' });
      expect(mockToast).toHaveBeenCalledWith({ title: 'items.createSuccess' });
      expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['items'] });
      expect(mockOnSuccess).toHaveBeenCalled();
      expect(mockOnCreateSuccess).toHaveBeenCalledWith({ id: 1, name: 'New Item' });
    });

    it('shows error toast on failure', async () => {
      const mockCreateFn = jest.fn().mockRejectedValue(new Error('Create failed'));

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            createFn: mockCreateFn,
          }),
        { wrapper }
      );

      act(() => {
        result.current.createMutation.mutate({ name: 'New Item' });
      });

      await waitFor(() => {
        expect(result.current.createMutation.isError).toBe(true);
      });

      expect(mockShowErrorToast).toHaveBeenCalledWith(
        'common.error',
        expect.any(Error),
        'common.failedToCreate'
      );
    });

    it('throws error if createFn not provided', async () => {
      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
          }),
        { wrapper }
      );

      act(() => {
        result.current.createMutation.mutate({ name: 'New Item' });
      });

      await waitFor(() => {
        expect(result.current.createMutation.isError).toBe(true);
      });
    });
  });

  describe('updateMutation', () => {
    it('calls updateFn and shows success toast on success', async () => {
      const mockUpdateFn = jest.fn().mockResolvedValue({ id: 1, name: 'Updated Item' });
      const mockOnSuccess = jest.fn();
      const mockOnUpdateSuccess = jest.fn();

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            updateFn: mockUpdateFn,
            onSuccess: mockOnSuccess,
            onUpdateSuccess: mockOnUpdateSuccess,
          }),
        { wrapper }
      );

      act(() => {
        result.current.updateMutation.mutate({ id: 1, data: { name: 'Updated Item' } });
      });

      await waitFor(() => {
        expect(result.current.updateMutation.isSuccess).toBe(true);
      });

      expect(mockUpdateFn).toHaveBeenCalledWith(1, { name: 'Updated Item' });
      expect(mockToast).toHaveBeenCalledWith({ title: 'items.updateSuccess' });
      expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['items'] });
      expect(mockOnSuccess).toHaveBeenCalled();
      expect(mockOnUpdateSuccess).toHaveBeenCalledWith({ id: 1, name: 'Updated Item' });
    });

    it('shows error toast on failure', async () => {
      const mockUpdateFn = jest.fn().mockRejectedValue(new Error('Update failed'));

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            updateFn: mockUpdateFn,
          }),
        { wrapper }
      );

      act(() => {
        result.current.updateMutation.mutate({ id: 1, data: { name: 'Updated' } });
      });

      await waitFor(() => {
        expect(result.current.updateMutation.isError).toBe(true);
      });

      expect(mockShowErrorToast).toHaveBeenCalledWith(
        'common.error',
        expect.any(Error),
        'common.failedToSave'
      );
    });
  });

  describe('deleteMutation', () => {
    it('calls deleteFn and shows success toast on success', async () => {
      const mockDeleteFn = jest.fn().mockResolvedValue(undefined);
      const mockOnDeleteSuccess = jest.fn();

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            deleteFn: mockDeleteFn,
            onDeleteSuccess: mockOnDeleteSuccess,
          }),
        { wrapper }
      );

      act(() => {
        result.current.deleteMutation.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.deleteMutation.isSuccess).toBe(true);
      });

      expect(mockDeleteFn).toHaveBeenCalledWith(1);
      expect(mockToast).toHaveBeenCalledWith({ title: 'items.deleteSuccess' });
      expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['items'] });
      expect(mockOnDeleteSuccess).toHaveBeenCalled();
    });

    it('shows error toast on failure', async () => {
      const mockDeleteFn = jest.fn().mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            deleteFn: mockDeleteFn,
          }),
        { wrapper }
      );

      act(() => {
        result.current.deleteMutation.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.deleteMutation.isError).toBe(true);
      });

      expect(mockShowErrorToast).toHaveBeenCalledWith(
        'common.error',
        expect.any(Error),
        'common.failedToDelete'
      );
    });
  });

  describe('isMutating', () => {
    it('returns true when any mutation is pending', async () => {
      let resolvePromise: (value: TestItem) => void;
      const mockCreateFn = jest.fn().mockImplementation(
        () =>
          new Promise<TestItem>((resolve) => {
            resolvePromise = resolve;
          })
      );

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            createFn: mockCreateFn,
          }),
        { wrapper }
      );

      expect(result.current.isMutating).toBe(false);

      act(() => {
        result.current.createMutation.mutate({ name: 'New Item' });
      });

      // Wait for mutation to be pending
      await waitFor(() => {
        expect(result.current.isMutating).toBe(true);
      });

      // Resolve the promise
      act(() => {
        resolvePromise!({ id: 1, name: 'Item' });
      });

      await waitFor(() => {
        expect(result.current.isMutating).toBe(false);
      });
    });
  });
});

describe('useCrudMutations — server field violations', () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createHookWrapper>;

  /** A rejection shaped like the API's problem document. */
  function rejection(params: Array<{ field: string; reason: string; rule?: string }>) {
    return {
      response: {
        data: {
          type: 'u',
          title: 'Validation failed',
          status: 400,
          code: 'validation',
          invalid_params: params.map((p) => ({ rule: 'required', ...p })),
        },
      },
    };
  }

  function form() {
    return {
      setError: jest.fn(),
      clearErrors: jest.fn(),
      getValues: jest.fn(() => ({ name: '' })),
    };
  }

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createHookWrapper(queryClient);
    mockToast.mockClear();
    mockShowErrorToast.mockClear();
  });

  it('marks the field the server named and keeps the toast quiet', async () => {
    const f = form();
    const { result } = renderHook(
      () =>
        useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
          resourceName: 'test',
          queryKey: ['test'],
          createFn: () => Promise.reject(rejection([{ field: 'name', reason: 'is required' }])),
          form: f as never,
        }),
      { wrapper }
    );

    act(() => {
      result.current.createMutation.mutate({ name: '' });
    });

    await waitFor(() => expect(f.setError).toHaveBeenCalled());
    expect(f.setError.mock.calls[0][0]).toBe('name');
    // The summary already lists it; a toast saying the same thing is noise.
    expect(mockShowErrorToast).not.toHaveBeenCalled();
  });

  it('still shows the toast when the rejection names no field', async () => {
    // A conflict or a network failure has nothing to mark, and silence would
    // leave a rejected submit looking like nothing happened.
    const f = form();
    const { result } = renderHook(
      () =>
        useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
          resourceName: 'test',
          queryKey: ['test'],
          createFn: () => Promise.reject(new Error('network down')),
          form: f as never,
        }),
      { wrapper }
    );

    act(() => {
      result.current.createMutation.mutate({ name: 'x' });
    });

    await waitFor(() => expect(mockShowErrorToast).toHaveBeenCalled());
    expect(f.setError).not.toHaveBeenCalled();
  });

  it('keeps a violation it cannot place, rather than dropping it', async () => {
    // The form has no such input, so nothing can be marked -- but the user still
    // has to be told, or the submit fails for a reason nobody states.
    const f = form();
    const { result } = renderHook(
      () =>
        useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
          resourceName: 'test',
          queryKey: ['test'],
          createFn: () =>
            Promise.reject(rejection([{ field: 'not_on_this_form', reason: 'is invalid' }])),
          form: f as never,
        }),
      { wrapper }
    );

    act(() => {
      result.current.createMutation.mutate({ name: 'x' });
    });

    await waitFor(() => expect(result.current.unmappedViolations).toHaveLength(1));
    expect(result.current.unmappedViolations[0].field).toBe('not_on_this_form');
    expect(mockShowErrorToast).not.toHaveBeenCalled();
  });

  it('does nothing to a form it was not given', async () => {
    // Delete-only callers pass no form; the toast stays their only report.
    const { result } = renderHook(
      () =>
        useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
          resourceName: 'test',
          queryKey: ['test'],
          createFn: () => Promise.reject(rejection([{ field: 'name', reason: 'is required' }])),
        }),
      { wrapper }
    );

    act(() => {
      result.current.createMutation.mutate({ name: '' });
    });

    await waitFor(() => expect(mockShowErrorToast).toHaveBeenCalled());
    expect(result.current.unmappedViolations).toHaveLength(0);
  });
});
